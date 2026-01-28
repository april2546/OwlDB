package contents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
	sse "github.com/RICE-COMP318-FALL24/owldb-p1group32/sse"
)

// For SSE logic
type WriteFlusher interface {
	http.ResponseWriter
	http.Flusher
}

// A Collection represents a collection of documents.
// It contains the name of the collection and the skiplist used for
// storing documents.
type Collection struct {
	Name      string                             // Name of the collection
	Documents skiplist.DBIndex[string, Document] // SkipList used to store the documents inside the collection
	// The key type for this SkipList is a string and the value type is a Document struct.
	Subscribers map[string]WriteFlusher
}

type ValidSchema interface {
	ValidateDocument(documentContent []byte) (bool, error)
}

// GetCollection retrieves a Collection from the given collectionList called collectionName.
// It searches the skiplist (collectionList) for the specified collection under collectionName.
// If the collection is found, it returns the corresponding Collection struct.
// If the collection is not found, it returns an error.
//
// The collectionList argument is a skiplist where the key is the name of the collection (string),
// and the value is a Collection struct.
//
// If found, the function returns the Collection and a nil error.
// If not found, the function returns a zero-value Collection and a non-nil error.
func GetCollection(collectionList skiplist.DBIndex[string, Collection], collectionName string, ctx context.Context, start string, end string) (Collection, error) {
	collection, found := collectionList.Find(collectionName)
	if !found {
		return collection, fmt.Errorf("failed to find database")
	}
	return collection, nil
}

// PutCollection tries to insert a new Collection into the given collectionList called collectionName.
// It uses an update check to ensure that a collection with the same name does not already exist.
// If the collection already exists, it returns false and an error.
// If the collection is successfully added, it returns true and a nil error.
//
// The collectionList argument is a skiplist where the key is the name of the collection (string),
// and the value is a Collection struct.
//
// The updateCheck function initializes a new Collection with the given collectionName and an empty skiplist
// for storing documents. If a collection with the same name exists, it fails with an error.
//
// Returns a boolean, either success (true) or failure (false), or an error if one occured.
func PutCollection(collectionList skiplist.DBIndex[string, Collection], collectionName string) (bool, error) {
	updateCheck := func(key string, currValue Collection, exists bool) (newValue Collection, err error) {
		if exists {
			return currValue, fmt.Errorf("database already exists")
		}
		newValue.Name = key
		newValue.Documents = skiplist.NewSkipList[string, Document]()
		return newValue, nil
	}

	return collectionList.Upsert(collectionName, updateCheck)
}

// DeleteDocument removes a given Collection from its respective skiplist. The inputs to this
// function are collectionList (a skiplist representing a list of collection) and collectionName (a string representing
// the collection that we are wanting to remove).
//
// The function attempts to remove the collection from its corresponding skiplist. If the removal fails, then
// the function returns false and an error. If the removal succeeds, then the function returns true and nil (no error).

func DeleteCollection(collectionList skiplist.DBIndex[string, Collection], collectionName string, subscriberHandler *sse.SubscriberHandler, fullPath string) (bool, error) {

	col, found := collectionList.Find(collectionName)
	if !found {
		return false, fmt.Errorf("could not find collection named %s", collectionName)
	}

	subscriberHandler.Notify(fullPath, "delete", strconv.Quote(fullPath))
	//subscriberHandler.Notify(fullPath, "delete", strconv.Quote(fullPath))
	col.HandleColDelete()

	// Now remove from the skiplist after handling delete with subscribers
	_, removed := collectionList.Remove(collectionName)
	if !removed {
		return false, fmt.Errorf("could not delete document named %s", collectionName)
	}
	return true, nil
}

// Notify subscribers
func (c *Collection) NotifySubscribers(eventName string, eventData string) {
	for _, wf := range c.Subscribers {
		var evt bytes.Buffer
		evt.WriteString(fmt.Sprintf("event: %s\ndata: %s\nid: %d\n\n", eventName, eventData, time.Now().UnixMilli()))
		wf.Write(evt.Bytes())
		wf.Flush()
	}
}

// Handle a document update in a collection
func (c *Collection) HandleDocumentUpdate(doc *Document) {
	metaJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		fmt.Printf("error serializing metadata: %v\n", err)
		return
	}
	eventData := fmt.Sprintf(`{"path":"%s","doc":%s,"meta":%s}`, doc.Path, strconv.Quote(string(doc.Content)), metaJSON)
	c.NotifySubscribers("update", eventData)
}

// Handle a document deletion in a collection
func (c *Collection) HandleDocumentDelete(docPath string) {
	eventData := strconv.Quote(docPath)
	c.NotifySubscribers("delete", eventData)
}

// Handle deletion of a collection
func (c *Collection) HandleColDelete() {
	eventData := strconv.Quote(c.Name)
	c.NotifySubscribers("delete", eventData)
}

// This struct represents a document in our database. The struct contains the name of the document,
// the path to the document, the document's contents, the Metadata associated with the document,
// the collections that are nested inside the document, and a map to the subscribers that are subscribed to the document.
type Document struct {
	Name        string // name of the document
	Path        string // path to the document in the database
	Content     []byte
	Metadata    Metadata                             // contains information about creation/modification of the document
	Collections skiplist.DBIndex[string, Collection] // skiplist holds the collections stored inside the document
	Subscribers map[string]WriteFlusher              // map of subscribers holding the path to the subscriber/client
}

// This struct holds the information of which user created the document and the time
// of creation/modification. This struct is used as a field in the document struct.
type Metadata struct {
	CreatedBy      string // string representing a user
	CreatedAt      int64  // integer representing the time the document was created
	LastModifiedBy string // string representing a user
	LastModifiedAt int64  // integer representing the last time the document was modified
}

// GetDocument retrieves a document from the given documentList called documentName.
// It searches the skiplist (documentList) for the specified document under documentName.
// If the document exists and is found, it returns the corresponding document struct and nil.
// If the document is not found, it returns an error.
//
// The documentList argument is a skiplist where the key is the name of the document (string),
// and the value is a Document struct.
func GetDocument(documentList skiplist.DBIndex[string, Document], documentName string) (Document, error) {
	document, found := documentList.Find(documentName)
	if !found {
		return document, fmt.Errorf("failed to find document")
	}

	// Metadata update
	return document, nil
}

// PutDocument tries to insert a new document into the given collectionList called documentName.
// The documentList argument is a skiplist where the key is the name of the collection (string),
// and the value is a document struct. The documentName argument is a string that is the name of
// the document that we are creating.
//
// PutDocument creates the name and the document content for the document, then validates that
// the content of the document matches the JSON schema that was provided witth the -s flag.
//
// Then, the function checks that there is not already a document with the given documentName if
// the mode for the request is set to "nooverwrite". If this occurs, then the function returns an error.
// If the mode is "overwrite", the function will update the metadata information accordingly.
//
// If a document with the given documentName doesn't already exist, then a new document and its
// metadata is created, a new skiplist for the Collections field is created, and the new document
// is Upserted into the given documentList.
//
// PutCollection returns true if successful, false if it fails, or an error.
func PutDocument(documentList skiplist.DBIndex[string, Document], documentName string, documentContent []byte, user string, mode string, schema ValidSchema) (bool, error) {

	updateCheck := func(key string, currValue Document, exists bool) (newValue Document, err error) {
		// Creating the name and content for the document
		newValue.Name = key
		newValue.Content = documentContent

		// Check that the document content matches the provided JSON Schema
		_, err = schema.ValidateDocument(newValue.Content)
		if err != nil {
			fmt.Errorf("Document content does not match the provided schema\n")
			return currValue, err
		}
		// Handle when in nooverwrite mode
		if exists && mode == "nooverwrite" {
			return currValue, fmt.Errorf("document exists and mode is nooverwrite")
		}
		// In overwrite or not specified mode, create/update like usual
		if exists {
			newValue.Metadata = Metadata{
				CreatedBy:      currValue.Metadata.CreatedBy,
				CreatedAt:      currValue.Metadata.CreatedAt,
				LastModifiedBy: user,
				LastModifiedAt: time.Now().Unix(),
			}
			// Handle the update for subscription
			currValue.HandleUpdate(documentContent, user)

			return newValue, nil
		}
		// Create a new document if doc doesn't exist
		newValue.Collections = skiplist.NewSkipList[string, Collection]()
		newValue.Metadata = Metadata{
			CreatedBy:      user,
			CreatedAt:      time.Now().Unix(),
			LastModifiedBy: user,              // Since it's a new document, the creator is also the last modifier
			LastModifiedAt: time.Now().Unix(), // Initial creation time is also the last modification time
		}
		// Notify subscribers about the new document
		newValue.NotifySubscribers("create", fmt.Sprintf(`{"path":"%s"}`, newValue.Path))
		return newValue, nil
	}
	return documentList.Upsert(documentName, updateCheck)
}

// DeleteDocument removes a given document from its respective skiplist. The inputs to this
// function are documentList (a skiplist representing a list of documents) and documentName (a string representing
// the document that we are wanting to remove).
//
// The function attempts to remove the document from its corresponding skiplist. If the removal fails, then
// the function returns false and an error. If the removal succeeds, then the function returns true and nil (no error).
func DeleteDocument(documentList skiplist.DBIndex[string, Document], documentName string, subscriberHandler *sse.SubscriberHandler, fullPath string) (bool, error) {
	doc, found := documentList.Find(documentName)
	if !found {
		return false, fmt.Errorf("could not find document named %s", documentName)
	}

	doc.HandleDocDelete()
	// Notify subscribers about the deletion
	subscriberHandler.Notify(fullPath, "delete", strconv.Quote(fullPath))

	// Now remove from the skiplist after handling subscribers
	_, removed := documentList.Remove(documentName)
	if !removed {
		return false, fmt.Errorf("could not delete document named %s", documentName)
	}
	return true, nil
}

// Notify subscribers
func (d *Document) NotifySubscribers(eventName string, eventData string) {
	for _, wf := range d.Subscribers {
		var evt bytes.Buffer
		evt.WriteString(fmt.Sprintf("event: %s\ndata: %s\nid: %d\n\n", eventName, eventData, time.Now().UnixMilli()))
		wf.Write(evt.Bytes())
		wf.Flush()
	}
}

// AddSubscriber adds a new subscriber to the document
func (d *Document) AddSubscriber(clientID string, wf WriteFlusher) {
	if d.Subscribers == nil {
		d.Subscribers = make(map[string]WriteFlusher)
	}
	d.Subscribers[clientID] = wf
}

// Handle an update to a doc
func (d *Document) HandleUpdate(newContent []byte, user string) {
	d.Content = newContent
	d.Metadata.LastModifiedBy = user
	d.Metadata.LastModifiedAt = time.Now().Unix()

	metaJSON, err := json.Marshal(d.Metadata)
	if err != nil {
		fmt.Printf("error serializing metadata: %v\n", err)
		return
	}

	eventData := fmt.Sprintf(`{"path":"%s","doc":%s,"meta":%s}`, d.Path, strconv.Quote(string(d.Content)), metaJSON)
	d.NotifySubscribers("update", eventData)
}

// Handle a deletion of a doc
func (d *Document) HandleDocDelete() {
	eventData := strconv.Quote(d.Path)
	d.NotifySubscribers("delete", eventData)
}
