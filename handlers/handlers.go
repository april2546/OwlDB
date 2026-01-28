// Package handlers initializes the databaseList where everything is stored,
// contains our main Handler that diverts requests, and contains our individual
// handlers for GET, PUT, POST, PATCH, and DELETE. This package also
// contains all of the helper functions needed for PATCH requests.
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/auth"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/contents"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/database"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/jsondata"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/sse"
)

// This struct holds all of the databases. It holds a skiplist,
// where each item in the skiplist represents an individual database,
// and a schema field which holds the valid schema that will be used
// to validate documents in the respective database.
type DatabaseList struct {
	databaseList      skiplist.DBIndex[string, database.Database]
	schema            Valid
	subscriberHandler *sse.SubscriberHandler
}

// This struct holds the informatio for a document response. It contains
// the name of the documet, its JSON contents, and its Metadata.
type DocumentResponse struct {
	Path string             `json:"path"`
	Doc  jsondata.JSONValue `json:"doc"`
	Meta contents.Metadata  `json:"meta"`
}

// This struct represents an operation for the PATCH request. It holds the operation for the request,
// the path for the request, and the JSON data being used for the request.
type PatchOperation struct {
	Op    string             `json:"op"`
	Path  string             `json:"path"`
	Value jsondata.JSONValue `json:"value"`
}

// PatchResponse struct enforces a fixed order for the JSON response
type PatchResponse struct {
	URI         string `json:"uri"`
	PatchFailed bool   `json:"patchFailed"`
	Message     string `json:"message"`
}

// New initializes a new list to store databases when starting the server. It returns
// an Database List containing an empty skiplist where we will store databases, and
// a schema passed through the -s flag.
func New(schema Valid, subscriberHandler *sse.SubscriberHandler) DatabaseList {

	return DatabaseList{
		databaseList:      skiplist.NewSkipList[string, database.Database](),
		schema:            schema,
		subscriberHandler: subscriberHandler,
	}
}

// type SSE interface {
// 	SSEHandler(w http.ResponseWriter, r *http.Request)
// }

// The ValidateDocument method in this interface is used to verify that the contents
// in a document match the schema that was provided
type Valid interface {
	ValidateDocument(documentContent []byte) (bool, error)
}

// ServeHTTP helps direct the incoming HTTP requests. It takes the request as input
// and then calls individual handlers depending on the type of request.
func (databaseList DatabaseList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	databaseList.V1Handler(w, r)
}

// V1Handler directs incoming HTTP requests to the correct handler method. It supports
// OPTIONS, GET, PUT, POST, DELETE, and PATCH.
// The function sets the CORS headers and logs the request details.
func (databaseList DatabaseList) V1Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Allow", "OPTIONS, GET, PUT, POST, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, PUT, POST, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	fmt.Println("Received request:", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		databaseList.GetHandler(w, r)
	case http.MethodPut:
		databaseList.PutHandler(w, r)
	case http.MethodPost:
		databaseList.PostHandler(w, r)
	case http.MethodDelete:
		databaseList.DeleteHandler(w, r)
	case http.MethodPatch:
		databaseList.PatchHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// This function is a helper used to return errors. It takes in a response writer, an
// HTTP status code, and the corresponding error message, and rewrites the error to the
// response writer.
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	errJson, _ := json.Marshal(message)
	w.Write(errJson)
}

// Helper function to create a DocumentResponse and append it to the response slice
func addDocumentResponse(path string, document contents.Document, documentResponses *[]DocumentResponse) {
	// Unmarshal document content from []byte
	var docContent map[string]interface{}
	err := json.Unmarshal(document.Content, &docContent)
	if err != nil {
		return
	}
	content, _ := jsondata.NewJSONValue(docContent)

	// Append the document response to the slice
	*documentResponses = append(*documentResponses, DocumentResponse{
		Path: path,
		Doc:  content,
		Meta: document.Metadata,
	})
}

// GetHandler handles requests trying to either GET a database or GET a document. The
// path to the document or database passed through the command line can be arbitrarily
// long including either databases, documents, or collections. GetHandler can either query
// everything inside a given database/document, or it can only retrieve a database's or document's
// contents within a given range.
func (databaseList DatabaseList) GetHandler(w http.ResponseWriter, r *http.Request) {

	// Setting the headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Checking that we don't have a slash in the middle of the paths
	pathEscaped := r.URL.EscapedPath()

	pathEscaped = strings.TrimPrefix(pathEscaped, "/v1/")
	// Trim ending /
	pathEscaped = strings.TrimSuffix(pathEscaped, "/")

	pathEscapedList := strings.Split(pathEscaped, "/")

	// Check for an empty path
	path := r.URL.Path

	// Trim /v1/
	path = strings.TrimPrefix(path, "/v1/")
	// Trim ending /
	path = strings.TrimSuffix(path, "/")

	pathList := strings.Split(path, "/")
	pathToReturn := "/" + pathList[len(pathList)-1]

	// Check for double slashes (//) in the path
	if strings.Contains(r.URL.Path, "//") {
		http.Error(w, "bad path: // not allowed", http.StatusBadRequest)
		return
	}
	// Case where databasename has a slash /
	if strings.Contains(pathEscapedList[0], "%2F") {
		respondWithError(w, http.StatusBadRequest, "Database should not contain /")
		return
	}

	// Get the interval if there is one
	interval := r.URL.Query().Get("interval")
	low := ""
	high := ""

	// Store the interval parameter if it exists
	if interval != "" {
		// Trim square brackets from the interval string
		trimmedInterval := strings.Trim(interval, "[]")

		// Split the interval string to separate low and high values
		intervalParts := strings.Split(trimmedInterval, ",")

		// Set the low/high values to whatever is passed in
		low = strings.TrimSpace(intervalParts[0])
		high = strings.TrimSpace(intervalParts[1])
	}

	// Extract Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header missing", http.StatusUnauthorized)
		return
	}

	// The Authorization header should be in the format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
		return
	}
	token := parts[1]
	mode := r.URL.Query().Get("mode")

	if strings.ToLower(mode) == "subscribe" {
		resource := path
		if resource == "" {
			http.Error(w, "resource missing", http.StatusBadRequest)
			return
		}
		databaseList.subscriberHandler.SSEHandler(w, r, resource, token)
	} else {
		//if not a subscribe get we set headers as a normal get request
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
	}
	var databaseFound database.Database
	var documentFound contents.Document
	var collectionFound contents.Collection
	var err error

	// Find the databases, documents, or collections
	for i, name := range pathList {
		if i == 0 {
			// First element will always be database
			//I want to Print what the skiplist looks like at this point
			databaseFound, err = database.GetDatabase(databaseList.databaseList, name, r.Context(), low, high)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Database does not exist")
				return
			}
		} else if i == 1 {
			documentFound, err = contents.GetDocument(databaseFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		} else if i%2 == 0 {
			// Even elements after first element will be collections
			collectionFound, err = contents.GetCollection(documentFound.Collections, name, r.Context(), low, high)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Collection does not exist")
				return
			}
		} else {
			// Odd elements will always be documents
			documentFound, err = contents.GetDocument(collectionFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		}
	}

	// If we've queried a full database or collection, we'll use the documents in `databaseFound.Documents` or `collectionFound.Documents`
	documentResponses := []DocumentResponse{}

	// db doc col

	if len(pathList) == 1 {
		// We queried a database; collect all documents within the database (which has already been filtered by `start` and `end` in GetDatabase)
		documents, _ := databaseFound.Documents.Query(r.Context(), "", "")
		for _, document := range documents {
			addDocumentResponse(pathToReturn, document, &documentResponses)
		}
	} else if len(pathList)%2 == 1 {
		// We queried a collection; collect all documents within the collection (already filtered)
		documents, _ := collectionFound.Documents.Query(r.Context(), "", "")
		for _, document := range documents {
			addDocumentResponse(pathToReturn, document, &documentResponses)
		}
	} else {
		// We queried a specific document; handle it directly
		addDocumentResponse(pathToReturn, documentFound, &documentResponses)
	}

	// Marshal the response into JSON and send it
	httpResponse, err := json.Marshal(documentResponses)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error marshaling json")
		return
	}

	w.Write(httpResponse)
}

// PutHandler handles requests trying to either PUT a database, document, or collection. This
// includes creating a new database/document/collection or replacing a current database/document/collection.
// The path to the document or database passed through the command line can be arbitrarily
// long including either databases, documents, or collection. For putting documents, the function calls a helper
// that validates the data inside the document against the provided schema. The response contains the URI of the newly created item.
func (databaseList DatabaseList) PutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	// Checking that we don't have a slash in the middle of the paths
	pathEscaped := r.URL.EscapedPath()

	pathEscaped = strings.TrimPrefix(pathEscaped, "/v1/")
	// Trim ending /
	pathEscaped = strings.TrimSuffix(pathEscaped, "/")

	pathEscapedList := strings.Split(pathEscaped, "/")

	// Trim /v1/
	path = strings.TrimPrefix(path, "/v1/")
	// Trim ending /
	path = strings.TrimSuffix(path, "/")

	pathList := strings.Split(path, "/")

	// Check for double slashes (//) in the path
	if strings.Contains(r.URL.Path, "//") {
		http.Error(w, "bad path: // not allowed", http.StatusBadRequest)
		return
	}

	// Case where databasename has a slash /
	if strings.Contains(pathEscapedList[0], "%2F") {
		respondWithError(w, http.StatusBadRequest, "Database should not contain /")
		return
	}

	// Get the mode if there is one for putting documents
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "overwrite"
	}

	var databaseFound database.Database
	var documentFound contents.Document
	var collectionFound contents.Collection
	var err error
	var documentExists bool

	// Find the databases, documents, or collections
	for i, name := range pathList {
		if i == len(pathList)-1 {
			break
		}
		if i == 0 {
			// First element will always be database
			databaseFound, err = database.GetDatabase(databaseList.databaseList, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Database does not exist")
				return
			}
		} else if i == 1 {
			documentFound, err = contents.GetDocument(databaseFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		} else if i%2 == 0 {
			// Even elements after first element will be collections
			collectionFound, err = contents.GetCollection(documentFound.Collections, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Collection does not exist")
				return
			}
		} else {
			// Odd elements will always be documents
			documentFound, err = contents.GetDocument(collectionFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		}
	}

	// Put the database, documents, or collections
	if len(pathList) == 1 {
		// We should have a database
		_, err := database.PutDatabase(databaseList.databaseList, pathList[len(pathList)-1], databaseList.subscriberHandler, path)
		if err != nil {
			http.Error(w, "unable to create database "+pathList[len(pathList)-1]+": exists", http.StatusBadRequest)
			return
		}
	} else if len(pathList) == 2 {
		// We should have a document
		username, _ := auth.UsernameFromContext(r.Context())

		// Read the document content from the request body
		var documentContent jsondata.JSONValue
		err = json.NewDecoder(r.Body).Decode(&documentContent)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Marshal the document content into []byte format
		contentBytes, err := json.Marshal(documentContent)
		if err != nil {
			http.Error(w, "Failed to process document content", http.StatusInternalServerError)
			return
		}

		//_, err = contents.PutDocument(databaseFound.Documents, pathList[len(pathList)-1], contentBytes, username, mode, databaseList.schema)
		// Pass the `subscriberHandler` to notify subscribers when document is updated
		_, err = contents.GetDocument(databaseFound.Documents, pathList[len(pathList)-1])
		documentExists = err == nil

		// Inserting the document into its respective document list and verifying that its contents match the provided JSON Schema
		_, err = contents.PutDocument(databaseFound.Documents, pathList[len(pathList)-1], contentBytes, username, mode, databaseList.schema)
		if err != nil {
			http.Error(w, "Document contents did not match provided JSON schema", http.StatusBadRequest)
		}
	} else if len(pathList)%2 == 0 {
		// We should have a document from an arbitrarily long path
		username, _ := auth.UsernameFromContext(r.Context())
		// Read the document content from the request body
		var documentContent jsondata.JSONValue
		err = json.NewDecoder(r.Body).Decode(&documentContent)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Marshal the document content into []byte format
		contentBytes, err := json.Marshal(documentContent)
		if err != nil {
			http.Error(w, "Failed to process document content", http.StatusInternalServerError)
			return
		}

		_, err = contents.GetDocument(collectionFound.Documents, pathList[len(pathList)-1])
		documentExists = err == nil

		_, err = contents.PutDocument(collectionFound.Documents, pathList[len(pathList)-1], contentBytes, username, mode, databaseList.schema)
		if err != nil {
			http.Error(w, "Failed to put document", http.StatusBadRequest)
			return
		}
	} else {
		// We should have a collection
		_, err := contents.PutCollection(documentFound.Collections, pathList[len(pathList)-1])
		if err != nil {
			http.Error(w, "unable to create collection "+pathList[len(pathList)-1]+": exists", http.StatusBadRequest)
			return
		}
	}

	databaseList.subscriberHandler.Notify(path, "update", fmt.Sprintf("{\"path\":\"%s\"}", path))

	// Return the URI (path) of the document
	if mode == "overwrite" || mode == "" {
		if documentExists {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusCreated)
		}
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	uriResponse, _ := json.MarshalIndent(map[string]string{
		"uri": string(r.URL.Path),
	}, "", "  ")

	_, err = w.Write(uriResponse) // Write response to the client
	if err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		return
	}
}

// PostHandler handles POST requests for creating new documents within a database or collection.
// This includes adding documents to a top-level database or to an existing collection within a document.
// The path passed through the request can be arbitrarily long, indicating nested collections or documents.
// The handler allows specifying a mode (overwrite) for handling existing documents.
// It generates a document name, handles the request body containing the document's content,
// and inserts the document into the appropriate database or collection. The response contains the URI of the newly created document.
func (databaseList DatabaseList) PostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path
	if path == "" {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
	}

	// Get the mode if there is one for putting documents
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "overwrite"
	}

	// Checking that we don't have a slash in the middle of the paths
	pathEscaped := r.URL.EscapedPath()

	pathEscaped = strings.TrimPrefix(pathEscaped, "/v1/")
	// Trim ending /
	pathEscaped = strings.TrimSuffix(pathEscaped, "/")

	pathEscapedList := strings.Split(pathEscaped, "/")

	// Trim /v1/
	path = strings.TrimPrefix(path, "/v1/")
	// Trim ending /
	path = strings.TrimSuffix(path, "/")

	pathList := strings.Split(path, "/")

	// Check for double slashes (//) in the path
	if strings.Contains(r.URL.Path, "//") {
		http.Error(w, "bad path: // not allowed", http.StatusBadRequest)
		return
	}
	// Case where databasename has a slash /
	if strings.Contains(pathEscapedList[0], "%2F") {
		respondWithError(w, http.StatusBadRequest, "Database should not contain /")
		return
	}

	var databaseFound database.Database
	var documentFound contents.Document
	var collectionFound contents.Collection
	var err error

	// Find the databases, documents, or collections
	for i, name := range pathList {
		if i == 0 {
			// First element will always be database
			databaseFound, err = database.GetDatabase(databaseList.databaseList, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Database does not exist")
				return
			}
		} else if i == 1 {
			documentFound, err = contents.GetDocument(databaseFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		} else if i%2 == 0 {
			// Even elements after first element will be collections
			collectionFound, err = contents.GetCollection(documentFound.Collections, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Collection does not exist")
				return
			}
		} else {
			// Odd elements will always be documents
			documentFound, err = contents.GetDocument(collectionFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		}
	}

	// Read the document from the request body
	doc, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Generate a unique name for the document
	docName := generateDocName()

	// Get username
	username, _ := auth.UsernameFromContext(r.Context())

	if len(pathList) == 1 {
		// We have a database
		contents.PutDocument(databaseFound.Documents, docName, doc, username, mode, databaseList.schema)
	} else {
		// We have a collection
		contents.PutDocument(collectionFound.Documents, docName, doc, username, mode, databaseList.schema)
	}

	// Create the response
	uriResponse, _ := json.MarshalIndent(map[string]string{
		"uri": string(r.URL.Path) + string(docName),
	}, "", "  ")

	// Write the response
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(http.StatusCreated)
	w.Write(uriResponse)
}

// generate a unique document name
func generateDocName() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("doc-%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// DeleteHandler handles DELETE requests removing the specified database, document, or collection from
// its respective list. The function returns a StatusNoContent status code upon successful completion.
func (databaseList DatabaseList) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Checking that we don't have a slash in the middle of the paths
	pathEscaped := r.URL.EscapedPath()

	pathEscaped = strings.TrimPrefix(pathEscaped, "/v1/")
	// Trim ending /
	pathEscaped = strings.TrimSuffix(pathEscaped, "/")

	pathEscapedList := strings.Split(pathEscaped, "/")

	// Check for an empty path
	path := r.URL.Path
	// Trim /v1/
	path = strings.TrimPrefix(path, "/v1/")
	// Trim ending /
	path = strings.TrimSuffix(path, "/")

	pathList := strings.Split(path, "/")

	// Check for double slashes (//) in the path
	if strings.Contains(r.URL.Path, "//") {
		http.Error(w, "bad path: // not allowed", http.StatusBadRequest)
		return
	}
	// Case where databasename has a slash /
	if strings.Contains(pathEscapedList[0], "%2F") {
		respondWithError(w, http.StatusBadRequest, "Database should not contain /")
		return
	}

	var databaseFound database.Database
	var documentFound contents.Document
	var collectionFound contents.Collection
	var err error

	// Find the databases, documents, or collections
	for i, name := range pathList {
		if i == 0 {
			// First element will always be database
			databaseFound, err = database.GetDatabase(databaseList.databaseList, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Database does not exist")
				return
			}
		} else if i == 1 {
			documentFound, err = contents.GetDocument(databaseFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		} else if i%2 == 0 {
			// Even elements after first element will be collections
			collectionFound, err = contents.GetCollection(documentFound.Collections, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Collection does not exist")
				return
			}
		} else {
			// Odd elements will always be documents
			documentFound, err = contents.GetDocument(collectionFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		}
	}

	if len(pathList) == 1 {
		// We queried a database; collect all documents within the database (which has already been filtered by `start` and `end` in GetDatabase)
		database.DeleteDatabase(databaseList.databaseList, databaseFound, databaseList.subscriberHandler, path)
		w.WriteHeader(http.StatusNoContent)
	} else if len(pathList) == 2 {
		// We queried a specific document; handle it directly
		contents.DeleteDocument(databaseFound.Documents, documentFound.Name, databaseList.subscriberHandler, path)
		// Return the URI (path) of the document
		w.WriteHeader(http.StatusNoContent)
		return
		//we are delteing a nested document
	} else if len(pathList)%2 == 0 {

		databaseFound, err = database.GetDatabase(databaseList.databaseList, pathList[0], r.Context(), "", "")
		if err != nil {
			http.Error(w, "Database in path not found", http.StatusNotFound)
			return
		}

		documentFound, err = contents.GetDocument(databaseFound.Documents, pathList[1])
		if err != nil {
			http.Error(w, "Document in path not found", http.StatusNotFound)
			return
		}

		for i := 2; i < len(pathList); i++ {
			if i%2 == 1 {
				documentFound, err = contents.GetDocument(collectionFound.Documents, pathList[i])
				if err != nil {
					http.Error(w, "Document in path not found", http.StatusNotFound)
					return
				}
			} else {
				collectionFound, err = contents.GetCollection(documentFound.Collections, pathList[i], r.Context(), "", "")
				if err != nil {
					http.Error(w, "Collection in path not found", http.StatusNotFound)
					return
				}
			}
		}

		contents.DeleteDocument(collectionFound.Documents, documentFound.Name, databaseList.subscriberHandler, path)
		// Return the URI (path) of the document
		w.WriteHeader(http.StatusNoContent)
		return

	} else {
		// We are trying to delete a collection
		databaseFound, err = database.GetDatabase(databaseList.databaseList, pathList[0], r.Context(), "", "")
		if err != nil {
			http.Error(w, "Database in path not found", http.StatusNotFound)
			return
		}

		// Check existence of the document where the collection is located
		documentFound, err = contents.GetDocument(databaseFound.Documents, pathList[1])
		if err != nil {
			http.Error(w, "Document in path not found", http.StatusNotFound)
			return
		}

		for i := 2; i < len(pathList); i++ {
			if i%2 == 1 {
				documentFound, err = contents.GetDocument(collectionFound.Documents, pathList[i])
				if err != nil {
					http.Error(w, "Document in path not found", http.StatusNotFound)
					return
				}
			} else {
				collectionFound, err = contents.GetCollection(documentFound.Collections, pathList[i], r.Context(), "", "")
				if err != nil {
					http.Error(w, "Collection in path not found", http.StatusNotFound)
					return
				}
			}
		}

		contents.DeleteCollection(documentFound.Collections, collectionFound.Name, databaseList.subscriberHandler, path)
	}
	w.WriteHeader(http.StatusNoContent)

}

// PatchHandler handles PATCH requests for modifying an existing database, document, or collection.
// It is used to apply partial updates to documents/collections, while making sure that the contents still
// match the given schema.
// This function first validates the request, identifies the database and document needed for the patch,
// and then applies the specified patch operations atomically.
func (databaseList DatabaseList) PatchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Checking that we don't have a slash in the middle of the paths
	pathEscaped := r.URL.EscapedPath()

	pathEscaped = strings.TrimPrefix(pathEscaped, "/v1/")
	// Trim ending /
	pathEscaped = strings.TrimSuffix(pathEscaped, "/")

	pathEscapedList := strings.Split(pathEscaped, "/")

	// Check for an empty path
	path := r.URL.Path

	// Trim /v1/
	path = strings.TrimPrefix(path, "/v1/")
	// Trim ending /
	path = strings.TrimSuffix(path, "/")

	pathList := strings.Split(path, "/")
	pathToReturn := "/" + pathList[len(pathList)-1]
	// Check for double slashes (//) in the path
	if strings.Contains(r.URL.Path, "//") {
		http.Error(w, "bad path: // not allowed", http.StatusBadRequest)
		return
	}
	// Case where databasename has a slash /
	if strings.Contains(pathEscapedList[0], "%2F") {
		respondWithError(w, http.StatusBadRequest, "Database should not contain /")
		return
	}

	if len(pathList) < 2 {
		respondWithError(w, http.StatusBadRequest, "Invalid path for PATCH request")
		return
	}

	var databaseFound database.Database
	var documentFound contents.Document
	var collectionFound contents.Collection
	var err error

	// Find the databases, documents, or collections
	for i, name := range pathList {
		// Case for getting a database
		if i == 0 {
			// First element will always be database
			databaseFound, err = database.GetDatabase(databaseList.databaseList, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Database does not exist")
				return
			}
			// Case for getting a top-level document
		} else if i == 1 {
			documentFound, err = contents.GetDocument(databaseFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
			// Case for getting a collection
		} else if i%2 == 0 {
			// Even elements after first element will be collections
			collectionFound, err = contents.GetCollection(documentFound.Collections, name, r.Context(), "", "")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Collection does not exist")
				return
			}
		} else {
			// Case for getting nested documents
			// Odd elements will always be documents
			documentFound, err = contents.GetDocument(collectionFound.Documents, name)
			if err != nil {
				respondWithError(w, http.StatusNotFound, "Document does not exist")
				return
			}
		}
	}

	// Read the patch operations from the request body
	var patchOps []PatchOperation
	err = json.NewDecoder(r.Body).Decode(&patchOps)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid patch request body")
		return
	}

	// Apply each patch operation atomically
	patchFailed := false
	var message string

	// Create a copy of the original document content
	originalContent := documentFound.Content

	//get user
	username, _ := auth.UsernameFromContext(r.Context())

	// Supported operations
	supportedOps := map[string]bool{"ArrayAdd": true, "ArrayRemove": true, "ObjectAdd": true}

	// Try applying patches
	for _, patch := range patchOps {
		// Check if operation type is valid
		if !supportedOps[patch.Op] {
			patchFailed = true
			message = fmt.Sprintf("Invalid operation type: %v", patch.Op)
			break
		}
		// If document is in a collection
		if len(pathList) > 2 {
			err := applyPatch(&documentFound, patch, collectionFound.Documents, username, databaseList.schema)
			if err != nil {
				patchFailed = true
				message = fmt.Sprintf("Patch failed: %v", err)
				break
			}
		} else {
			err := applyPatch(&documentFound, patch, databaseFound.Documents, username, databaseList.schema)
			if err != nil {
				patchFailed = true
				message = fmt.Sprintf("Patch failed: %v", err)
				break
			}
		}
	}

	if patchFailed {
		// If patch failed, revert to original content
		documentFound.Content = originalContent
	} else {
		message = "patch applied"
	}

	// Create the response struct
	response := PatchResponse{
		URI:         pathToReturn,
		PatchFailed: patchFailed,
		Message:     message,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// This helper function checks that the provided patch operation from the request body is valid.
// It will direct the request to the proper helpers if the given patch operation is valid or it will
// throw an error if the provided patch operation is not valid.
func applyPatch(document *contents.Document, patch PatchOperation, documentList skiplist.DBIndex[string, contents.Document], user string, schema Valid) error {
	switch patch.Op {
	case "ArrayAdd":
		fmt.Println("were adding an array")
		return applyArrayAdd(document, patch.Path, patch.Value, documentList, user, schema)
	case "ArrayRemove":
		fmt.Println("were removing an array")
		return applyArrayRemove(document, patch.Path, patch.Value, documentList, user, schema)
	case "ObjectAdd":
		fmt.Println("were adding an object")
		return applyObjectAdd(document, patch.Path, patch.Value, documentList, user, schema)
	default:
		return fmt.Errorf("unsupported patch operation: %s", patch.Op)
	}
}

// This interface implements the visitor map that we will use in our PATCH requests
type Visitor interface {
	VisitMap(map[string]interface{}) error
}

// This helper function is used to add content to an Object in the document that we are patching on.
// The content from the request is unmarshalled, the visitor map is applied, and upon success the
// content is re-marshalled and added to the document.
func applyObjectAdd(document *contents.Document, path string, value jsondata.JSONValue, documentList skiplist.DBIndex[string, contents.Document], user string, schema Valid) error {
	var jsonContent map[string]interface{}

	// Unmarshal the document.Content (which is []byte) into a map[string]interface{}
	err := json.Unmarshal(document.Content, &jsonContent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal document content: %w", err)
	}

	// Create the visitor for adding an object
	visitor := &ObjectAddVisitor{
		Path:  path,
		Value: value,
	}

	// Apply the visitor to the JSON structure
	err = visitor.VisitMap(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to add object: %w", err)
	}

	// Re-marshal the modified jsonContent into document.Content
	document.Content, err = json.Marshal(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to marshal modified document content: %w", err)
	}

	// Upsert the modified document back into the skiplist
	_, err = contents.PutDocument(documentList, document.Name, document.Content, user, "overwrite", schema)
	if err != nil {
		return fmt.Errorf("failed to update document in skiplist: %w", err)
	}

	return nil
}

// This helper function is used to add content to an Array in the document that we are patching on.
// The content from the request is unmarshalled, the visitor map is applied, and upon success the
// content is re-marshalled and added to the document.
func applyArrayAdd(document *contents.Document, path string, value jsondata.JSONValue, documentList skiplist.DBIndex[string, contents.Document], user string, schema Valid) error {
	var jsonContent map[string]interface{}

	// Unmarshal the document.Content (which is []byte) into a map[string]interface{}
	err := json.Unmarshal(document.Content, &jsonContent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal document content: %w", err)
	}

	// Create the visitor for adding an element to an array
	visitor := &ArrayAddVisitor{
		Path:  path,
		Value: value,
	}

	// Apply the visitor to the JSON structure
	err = visitor.VisitMap(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to add array element: %w", err)
	}

	// Re-marshal the modified jsonContent into document.Content
	document.Content, err = json.Marshal(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to marshal modified document content: %w", err)
	}

	// Upsert the modified document back into the skiplist
	_, err = contents.PutDocument(documentList, document.Name, document.Content, user, "overwrite", schema)
	if err != nil {
		return fmt.Errorf("failed to update document in skiplist: %w", err)
	}

	return nil
}

// This helper function is used to remove content from an Array in the document that we are patching on.
// The content from the request is unmarshalled, the visitor map is applied, and upon success the
// content is re-marshalled and added to the document.
func applyArrayRemove(document *contents.Document, path string, value jsondata.JSONValue, documentList skiplist.DBIndex[string, contents.Document], user string, schema Valid) error {
	var jsonContent map[string]interface{}

	// Unmarshal the document.Content (which is []byte) into a map[string]interface{}
	err := json.Unmarshal(document.Content, &jsonContent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal document content: %w", err)
	}

	// Create the visitor for removing an element from an array
	visitor := &ArrayRemoveVisitor{
		Path:  path,
		Value: value,
	}

	// Apply the visitor to the JSON structure
	err = visitor.VisitMap(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to remove array element: %w", err)
	}

	// Re-marshal the modified jsonContent into document.Content
	document.Content, err = json.Marshal(jsonContent)
	if err != nil {
		return fmt.Errorf("failed to marshal modified document content: %w", err)
	}

	// Upsert the modified document back into the skiplist
	_, err = contents.PutDocument(documentList, document.Name, document.Content, user, "overwrite", schema)
	if err != nil {
		return fmt.Errorf("failed to update document in skiplist: %w", err)
	}

	return nil
}

type ObjectAddVisitor struct {
	Path  string
	Value interface{}
}

// This function is a helper for the applyObjectAdd for our PATCH handler. It iterates through the parts
// of the path making sure that everything exists, and then accesses the map or array element depending on the
// PATCH request. Once it finds the spot in the array or map where we are adding content, it adds the
// content into the final key in the provided path.
func (v *ObjectAddVisitor) VisitMap(data map[string]interface{}) error {
	pathParts := strings.Split(strings.TrimPrefix(v.Path, "/"), "/")

	// Traverse the JSON structure according to the path
	curr := data
	for i := 0; i < len(pathParts)-1; i++ {
		part := decodeJSONPointerToken(pathParts[i])

		// // If the current value is not a map, the path is invalid
		// if next, ok := curr[part].(map[string]interface{}); ok {
		// 	curr = next
		// } else {
		// 	return fmt.Errorf("path %s does not resolve to an object", v.Path)
		// }
		// If the current value is a map, access the map element
		if nextMap, ok := curr[part].(map[string]interface{}); ok {
			curr = nextMap
		} else if nextArray, ok := curr[part].([]interface{}); ok {
			// Handle array indexing
			index, err := strconv.Atoi(pathParts[i+1])
			if err != nil || index < 0 || index >= len(nextArray) {
				return fmt.Errorf("invalid array index at path %s", v.Path)
			}

			// Ensure the element at the index is a map
			if nextMap, ok := nextArray[index].(map[string]interface{}); ok {
				curr = nextMap
			} else {
				return fmt.Errorf("element at path %s is not a valid object", v.Path)
			}

			i++ // Skip the next part as we are processing the array index
		} else {
			return fmt.Errorf("path %s does not resolve to an object or array", v.Path)
		}
	}

	// Get the final key in the path and add the new object
	finalKey := decodeJSONPointerToken(pathParts[len(pathParts)-1])
	curr[finalKey] = v.Value
	return nil
}

type ArrayAddVisitor struct {
	Path  string
	Value interface{}
}

// This function is a helper for the applyArrayAdd for our PATCH handler. It iterates through the parts
// of the path making sure that everything exists, and then accesses the map or array element depending on the
// PATCH request. Once it finds the spot in the array or map where we are adding content, it adds the
// content into the final key in the provided path.
func (v *ArrayAddVisitor) VisitMap(data map[string]interface{}) error {
	pathParts := strings.Split(strings.TrimPrefix(v.Path, "/"), "/")

	// Traverse the JSON structure according to the path
	curr := data
	for i := 0; i < len(pathParts)-1; i++ {
		part := decodeJSONPointerToken(pathParts[i])

		if nextMap, ok := curr[part].(map[string]interface{}); ok {
			curr = nextMap
		} else if nextArray, ok := curr[part].([]interface{}); ok {
			// Handle array indexing
			index, err := strconv.Atoi(pathParts[i+1])
			if err != nil || index < 0 || index >= len(nextArray) {
				return fmt.Errorf("invalid array index at path %s", v.Path)
			}

			// Ensure the element at the index is a map[string]interface{}
			if nextMap, ok := nextArray[index].(map[string]interface{}); ok {
				curr = nextMap
			} else {
				return fmt.Errorf("element at path %s is not a valid object", v.Path)
			}
			i++ // Skip the next part as we are processing the array index
		} else {
			return fmt.Errorf("path %s does not resolve to an object or array", v.Path)
		}
	}

	// Get the final key in the path and check if it's an array
	finalKey := decodeJSONPointerToken(pathParts[len(pathParts)-1])
	if arr, ok := curr[finalKey].([]interface{}); ok {
		// Add the value to the array
		curr[finalKey] = append(arr, v.Value)
	} else {
		return fmt.Errorf("value at path %s is not an array", v.Path)
	}

	return nil
}

type ArrayRemoveVisitor struct {
	Path  string
	Value jsondata.JSONValue
}

// This function is a helper for the applyArrayRemove for our PATCH handler. It iterates through the parts
// of the path making sure that everything exists, and then accesses the map or array element depending on the
// PATCH request. Once it finds the spot in the array or map where we are adding content, it adds the
// content into the final key in the provided path.
func (v *ArrayRemoveVisitor) VisitMap(data map[string]interface{}) error {
	pathParts := strings.Split(strings.TrimPrefix(v.Path, "/"), "/")

	// Traverse the JSON structure according to the path
	curr := data
	for i := 0; i < len(pathParts)-1; i++ {
		part := decodeJSONPointerToken(pathParts[i])

		// // If the current value is not a map, the path is invalid
		// if next, ok := curr[part].(map[string]interface{}); ok {
		// 	curr = next
		// } else {
		// 	return fmt.Errorf("path %s does not resolve to an array", v.Path)
		// }
		// If the current value is a map, access the map element
		if nextMap, ok := curr[part].(map[string]interface{}); ok {
			curr = nextMap
		} else if nextArray, ok := curr[part].([]interface{}); ok {
			// If the current value is an array, handle array indexing
			index, err := strconv.Atoi(pathParts[i+1])
			if err != nil || index < 0 || index >= len(nextArray) {
				return fmt.Errorf("invalid array index at path %s", v.Path)
			}
			// Ensure the element at the index is a map[string]interface{}
			if nextMap, ok := nextArray[index].(map[string]interface{}); ok {
				curr = nextMap
			} else {
				return fmt.Errorf("element at path %s is not a valid object", v.Path)
			}
			i++ // Skip the next part as we are processing the array index
		} else {
			return fmt.Errorf("path %s does not resolve to an object or array", v.Path)
		}
	}

	// Get the final key in the path and check if it's an array
	finalKey := decodeJSONPointerToken(pathParts[len(pathParts)-1])

	if arr, ok := curr[finalKey].([]interface{}); ok {
		// Iterate over array elements and compare them with v.Value using JSONValue.Equal
		for i, elem := range arr {
			elemJSONValue, err := jsondata.NewJSONValue(elem) // Convert element to JSONValue for comparison
			if err != nil {
				return fmt.Errorf("failed to convert element to JSONValue: %w", err)
			}

			if v.Value.Equal(elemJSONValue) {
				// Remove element from array if it matches v.Value
				curr[finalKey] = append(arr[:i], arr[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("value not found in array at path %s", v.Path)
	} else {
		return fmt.Errorf("value at path %s is not an array", v.Path)
	}
}

// This is a helper function for the VisitMap functions that helps convert characters.
func decodeJSONPointerToken(token string) string {
	// Replace ~1 with / and ~0 with ~ according to the JSON Pointer specification
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")
	return token
}
