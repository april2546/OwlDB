package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/contents"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/database"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/jsondata"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
)

type testHandler struct {
	r            *http.Request
	w            *httptest.ResponseRecorder
	expectedCode int
	expectError  bool
}

func TestV1Handler(t *testing.T) {
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
	}

	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}

	testCases := []testHandler{
		// one test per method
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/db1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodOptions, "/v1/db1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/db1//doc1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/db1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/db1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNoContent,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodConnect, "/v1/db1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusMethodNotAllowed,
			expectError:  true,
		},
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/nonexisting/", strings.NewReader("{\"esther\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodPatch, "/v1/fakeDB/doc1", strings.NewReader("[{\"op\": \"ArrayAdd\", \"path\": \"/field1\", \"value\": { \"key\": \"value\"}}]")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}
	for _, tc := range testCases {
		request := tc.r
		testDBList.V1Handler(tc.w, request)

		if tc.expectedCode != tc.w.Code {
			//errorMessage := fmt.Sprintf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
			t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
			return
		}

		if tc.expectError == false {
			if tc.w.Code == http.StatusMethodNotAllowed {
				//errorMessage := fmt.Sprintf("Test failed, status code 405 was expected but received %d", tc.w.Code)
				t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
				return
			}
		}
		if tc.expectError == true {
			if tc.w.Code == http.StatusOK {
				//errorMessage := fmt.Sprintf("Test failed, status code 400 was expected but received %d", tc.w.Code)
				t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
				return
			}
		}
	}
}

func TestGetHandler(t *testing.T) {
	ctx := context.TODO()
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
	}
	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}
	testDB := database.Database{
		Name:      "testDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc1 := contents.Document{
		Name:    "testDoc1",
		Path:    "/v1/testDB/testDoc1",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "julia",
			CreatedAt:      12,
			LastModifiedBy: "julia",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testDoc2 := contents.Document{
		Name:    "testDoc2",
		Path:    "/v1/testDB/testDoc2",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testCollection1 := contents.Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc3 := contents.Document{
		Name:    "testDoc3",
		Path:    "/v1/testDB/testDoc2/testCollection1/testDoc3",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}

	_, err := testDBList.databaseList.Upsert(testDB.Name, func(key string, currValue database.Database, exists bool) (database.Database, error) {
		return testDB, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDB: %v", err)
		return
	}
	_, err = testDB.Documents.Upsert(testDoc1.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testDB.Documents.Upsert(testDoc2.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
		return
	}
	_, err = testDoc2.Collections.Upsert(testCollection1.Name, func(key string, currValue contents.Collection, exists bool) (contents.Collection, error) {
		return testCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err)
		return
	}
	_, err = testCollection1.Documents.Upsert(testDoc3.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
		return
	}
	_, err1 := database.GetDatabase(testDBList.databaseList, "testDB", ctx, "", "")
	if err1 != nil {
		fmt.Printf("upserting the stuff in the test database failed")
		return
	} else {
		fmt.Print("ALL THE UPSERTING WORKED")
	}
	testCases := []testHandler{
		// Getting an existing database
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		// Getting an existing document
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc1", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		// Getting another existing document
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		// Getting an existing collection
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2/testCollection1/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		// Getting an existing nested collection
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2/testCollection1/testDoc3", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusOK,
			expectError:  false,
		},
		// Getting a non existing database
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/t", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// weird path test
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2/testCol/lection1/testDoc3", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// another weird path test
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB7/testDoc2/testCol/lection1/testDoc3", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// another slash in the middle of the path
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/tes/tDB7", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// double slash at the end of a database path
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB7//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// double slash at the end of a document path
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB7/testDoc1//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// double slash at the end of a collection path
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB7/testDoc1/testCollection1//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// non existing document
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2/testCollection1/testDoc8", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// non existing items in the path
		{
			r:            httptest.NewRequest(http.MethodGet, "/v1/testDB/testDoc2/testCollection4/testDoc3", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.r.Method, func(t *testing.T) {
			request := tc.r
			testDBList.GetHandler(tc.w, request)

			if tc.expectedCode != tc.w.Code {
				//errorMessage := fmt.Sprintf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				return
			}
			if tc.expectError == false {
				if tc.w.Code == http.StatusMethodNotAllowed {
					//errorMessage := fmt.Sprintf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					return
				}
			} else {
				if tc.w.Code == http.StatusOK {
					//errorMessage := fmt.Sprintf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					return
				}
			}
		})
	}
}

func TestPutHandler(t *testing.T) {
	ctx := context.TODO()
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
	}
	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}
	existingtestDB := database.Database{
		Name:      "existingtestDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}

	existingtestDoc2 := contents.Document{
		Name:    "existingtestDoc2",
		Path:    "/v1/existingtestDB/existingtestDoc2",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	existingtestCollection1 := contents.Collection{
		Name:      "existingtestCollection1",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	existingtestDoc3 := contents.Document{
		Name:    "existingtestDoc3",
		Path:    "/v1/existingtestDB/existingtestDoc2/existingtestCollection1/existingtestDoc3",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}

	_, err := testDBList.databaseList.Upsert(existingtestDB.Name, func(key string, currValue database.Database, exists bool) (database.Database, error) {
		return existingtestDB, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert existingtestDB: %v", err)
		return
	}
	_, err = existingtestDB.Documents.Upsert(existingtestDoc2.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return existingtestDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert existingtestDoc2: %v", err)
		return
	}
	_, err = existingtestDoc2.Collections.Upsert(existingtestCollection1.Name, func(key string, currValue contents.Collection, exists bool) (contents.Collection, error) {
		return existingtestCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert existingtestCollection1: %v", err)
		return
	}
	_, err = existingtestCollection1.Documents.Upsert(existingtestDoc3.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return existingtestDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert existingtestDoc3: %v", err)
		return
	}
	_, err1 := database.GetDatabase(testDBList.databaseList, "existingtestDB", ctx, "", "")
	if err1 != nil {
		fmt.Printf("upserting the stuff in the test database failed %v", err1)
		return
	} else {
		fmt.Print("ALL THE UPSERTING WORKED")
	}
	type testPutHandler struct {
		r                     *http.Request
		w                     *httptest.ResponseRecorder
		expectedCode          int
		expectError           bool
		ifDoc                 bool
		homeDb                database.Database
		homeCollection        contents.Collection
		nested                bool
		expectedGetDocContent []byte
		docName               string
		getPath               string
	}

	testCases := []testPutHandler{
		// putting a valid database
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
			ifDoc:        false,
			getPath:      "/v1/testDB",
		},
		// putting another database
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB2", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
			ifDoc:        false,
			getPath:      "/v1/testDB2",
		},
		// putting a document inside the preexisting database
		{
			r:                     httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/testDoc1", strings.NewReader("{\"julia\":2}")),
			w:                     httptest.NewRecorder(),
			expectedCode:          http.StatusCreated,
			expectError:           false,
			ifDoc:                 true,
			homeDb:                existingtestDB,
			nested:                false,
			expectedGetDocContent: []byte("{\"julia\":2}"),
			docName:               "testDoc1",
			getPath:               "/v1/existingtestDB/testDoc1",
		},
		// putting a document inside a preexisting database
		{
			r:                     httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/testDoc2", strings.NewReader("{\"april\":2}")),
			w:                     httptest.NewRecorder(),
			expectedCode:          http.StatusCreated,
			expectError:           false,
			ifDoc:                 true,
			homeDb:                existingtestDB,
			nested:                false,
			expectedGetDocContent: []byte("{\"april\":2}"),
			docName:               "testDoc2",
			getPath:               "/v1/existingtestDB/testDoc2",
		},
		// putting a document inside a non existing database
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB3/testDoc3", strings.NewReader("{\"prop\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// putting a collection inside an existing document
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/existingtestDoc2/testCollection1/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
			ifDoc:        false,
			getPath:      "/v1/existingtestDB/existingtestDoc2/testCollection1/",
		},
		// putting a collection inside a non existing document
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDoc5/testCollection2/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// putting a collection inside a non existing database
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB3/testDoc2/testCollection2/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// putting a document inside an existing collection
		{
			r:                     httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/existingtestDoc2/existingtestCollection1/testDoc7", strings.NewReader("{\"prop\":2}")),
			w:                     httptest.NewRecorder(),
			expectedCode:          http.StatusCreated,
			expectError:           false,
			ifDoc:                 true,
			homeDb:                existingtestDB,
			nested:                true,
			homeCollection:        existingtestCollection1,
			expectedGetDocContent: []byte("{\"prop\":2}"),
			docName:               "testDoc7",
			getPath:               "/v1/existingtestDB/existingtestDoc2/existingtestCollection1/testDoc7",
		},
		// putting a document inside a non existing collection
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDoc2/testCollection5/testDoc8", strings.NewReader("{\"prop\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// putting a database with double slashes in the path
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// putting a document with double slashes at the end of the path
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDocument88//", strings.NewReader("{\"prop\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// same as above but with collection
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDoc1/testCOllection4//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// more double slashes with a nested document
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDoc2/testCollection1/anotherdoc//", strings.NewReader("{\"prop\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// These are the schema tests
		// contents don't match schema
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDocwithSchema", strings.NewReader("{\"name\":\"julia\"}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/testDB/testDocwithBadContents", strings.NewReader("")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/existingtestDoc2/testCollection1/AnotherSchemaTest", strings.NewReader("more bad document body")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// content matches the schema
		{
			r:            httptest.NewRequest(http.MethodPut, "/v1/existingtestDB/existingtestDoc2/testCollection1/CorrectSchemaTest", strings.NewReader("{\"name\":\"julia\",\"age\":12}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
	}
	i := 0
	for _, tc := range testCases {
		t.Run(tc.r.Method, func(t *testing.T) {
			testDBList.PutHandler(tc.w, tc.r)

			if tc.expectedCode != tc.w.Code {
				fmt.Printf("ERROR IS %s", tc.w.Body)
				t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				return
			}
			if tc.expectError == false {
				if tc.w.Code == http.StatusMethodNotAllowed {
					t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					return
				}
				var resultDoc contents.Document
				if tc.ifDoc == true {
					if tc.nested == true {
						resultDoc, err = contents.GetDocument(tc.homeCollection.Documents, tc.docName)
						if err != nil {
							t.Fatalf("Test case failed: could not retrieve document contents after put")
							return
						}
					} else {
						resultDoc, err = contents.GetDocument(tc.homeDb.Documents, tc.docName)
						if err != nil {
							t.Fatalf("Test case failed: could not retrieve document contents after put")
							return
						}
					}
					for i := 0; i < len(resultDoc.Content); i++ {
						if resultDoc.Content[i] != tc.expectedGetDocContent[i] {
							t.Fatalf("Test case failed: expected contents do not match resulting contents.\n Expected: %s\n, Resulting: %s", string(tc.expectedGetDocContent), string(resultDoc.Content))
							return
						}
					}
				}
			} else {
				if tc.w.Code == http.StatusOK {
					t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					return
				}
			}
		})
		i += 1
		if i == (len(testCases) - 5) {
			nameAgeSchema, err2 := jsondata.New("../schema1.json")
			if err2 != nil {
				t.Fatalf("Test schema could not be successfully created")
				return
			}
			testDBList.schema = &nameAgeSchema
		}
	}
}

func TestDeletetHandler(t *testing.T) {
	ctx := context.TODO()
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
	}
	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}
	testDB := database.Database{
		Name:      "testDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc1 := contents.Document{
		Name:    "testDoc1",
		Path:    "/v1/testDB/testDoc1",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "julia",
			CreatedAt:      12,
			LastModifiedBy: "julia",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testDoc2 := contents.Document{
		Name:    "testDoc2",
		Path:    "/v1/testDB/testDoc2",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testCollection1 := contents.Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc3 := contents.Document{
		Name:    "testDoc3",
		Path:    "/v1/testDB/testDoc2/testCollection1/testDoc3",
		Content: []byte("Test document"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}

	_, err := testDBList.databaseList.Upsert(testDB.Name, func(key string, currValue database.Database, exists bool) (database.Database, error) {
		return testDB, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDB: %v", err)
	}
	_, err = testDB.Documents.Upsert(testDoc1.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testDB.Documents.Upsert(testDoc2.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testDoc2.Collections.Upsert(testCollection1.Name, func(key string, currValue contents.Collection, exists bool) (contents.Collection, error) {
		return testCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err)
	}
	_, err = testCollection1.Documents.Upsert(testDoc3.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
	}
	/// testDB/testDoc2/testCollection1/testDoc3
	_, err1 := database.GetDatabase(testDBList.databaseList, "testDB", ctx, "", "")
	if err1 != nil {
		fmt.Printf("upserting the stuff in the test database failed")
		return
	} else {
		fmt.Print("ALL THE UPSERTING WORKED")
	}
	testCases := []testHandler{
		// deleting an existing document
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc2/testCollection1/testDoc3", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNoContent,
			expectError:  false,
		},
		// deleting a non existent document
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc2/testCollection1/testDoc4", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// deleting an existing collection
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc2/testCollection1/", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNoContent,
			expectError:  false,
		},
		// deleting a nonexisting collection
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc1/testCollection2", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// deleting a nonexisting document
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc9", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// deleting an existing document
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc2", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNoContent,
			expectError:  false,
		},
		// deleting an existing database
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNoContent,
			expectError:  false,
		},
		// deleting a non existing database
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDBnonExisting", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// deleting collection with bad path
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc1/testCollection2//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// deleting document with bad path
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB/testDoc2//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// deleting database with bad path
		{
			r:            httptest.NewRequest(http.MethodDelete, "/v1/testDB//", nil),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.r.Method, func(t *testing.T) {
			request := tc.r
			testDBList.DeleteHandler(tc.w, request)

			if tc.expectedCode != tc.w.Code {
				//errorMessage := fmt.Sprintf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				return
			}
			if tc.expectError == false {
				if tc.w.Code == http.StatusMethodNotAllowed {
					//errorMessage := fmt.Sprintf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					return
				}
			}
			if tc.expectError == true {
				if tc.w.Code == http.StatusOK {
					//errorMessage := fmt.Sprintf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					return
				}
			}
		})
	}
}

func TestPostHandler(t *testing.T) {
	ctx := context.TODO()
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
		return
	}
	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}
	testDB := database.Database{
		Name:      "testDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc1 := contents.Document{
		Name:    "testDoc1",
		Path:    "/v1/testDB/testDoc1",
		Content: []byte("{\"julia\":2}"),
		Metadata: contents.Metadata{
			CreatedBy:      "julia",
			CreatedAt:      12,
			LastModifiedBy: "julia",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testDoc2 := contents.Document{
		Name:    "testDoc2",
		Path:    "/v1/testDB/testDoc2",
		Content: []byte("{\"april\":2}"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testCollection1 := contents.Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}

	_, err := testDBList.databaseList.Upsert(testDB.Name, func(key string, currValue database.Database, exists bool) (database.Database, error) {
		return testDB, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDB: %v", err)
		return
	}
	_, err = testDB.Documents.Upsert(testDoc1.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
		return
	}
	_, err = testDB.Documents.Upsert(testDoc2.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
		return
	}
	_, err = testDoc2.Collections.Upsert(testCollection1.Name, func(key string, currValue contents.Collection, exists bool) (contents.Collection, error) {
		return testCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err)
		return
	}

	_, err1 := database.GetDatabase(testDBList.databaseList, "testDB", ctx, "", "")
	if err1 != nil {
		fmt.Printf("upserting the stuff in the test database failed")
		return
	} else {
		fmt.Print("ALL THE UPSERTING WORKED")
	}
	testCases := []testHandler{
		// posting in the database
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/", strings.NewReader("{\"julia\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		// posting in a non existent database
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/nonExistingDB/", strings.NewReader("{\"julia\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// posting in an existing collection
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/testDoc2/testCollection1/", strings.NewReader("{\"april\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		// posting in a non existing collection
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/testDoc2/testCollection2/", strings.NewReader("{\"april\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  false,
		},
		// posting in a path with non existing elements
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/testDoc4/testCollection1/", strings.NewReader("{\"april\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusNotFound,
			expectError:  true,
		},
		// testing double slashes in a database and a collection
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB//", strings.NewReader("{\"esther\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/testDoc2//testCollection1", strings.NewReader("{\"esther\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
		// testing with a schema
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/", strings.NewReader("{\"name\":\"julia\",\"age\":12}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusCreated,
			expectError:  false,
		},
		{
			r:            httptest.NewRequest(http.MethodPost, "/v1/testDB/testDoc2//testCollection1", strings.NewReader("{\"esther\":2}")),
			w:            httptest.NewRecorder(),
			expectedCode: http.StatusBadRequest,
			expectError:  true,
		},
	}
	i := 0
	for _, tc := range testCases {
		t.Run(tc.r.Method, func(t *testing.T) {
			request := tc.r
			testDBList.PostHandler(tc.w, request)

			if tc.expectedCode != tc.w.Code {
				//errorMessage := fmt.Sprintf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				return
			}
			if tc.expectError == false {
				if tc.w.Code == http.StatusMethodNotAllowed {
					//errorMessage := fmt.Sprintf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					return
				}
			}
			if tc.expectError == true {
				if tc.w.Code == http.StatusOK {
					//errorMessage := fmt.Sprintf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					return
				}
			}
		})
		i += 1
		if i == (len(testCases) - 3) {
			nameAgeSchema, err2 := jsondata.New("../schema1.json")
			if err2 != nil {
				t.Fatalf("Test schema could not be successfully created")
				return
			}
			testDBList.schema = &nameAgeSchema
		}
	}
}

func TestPatchHandler(t *testing.T) {
	ctx := context.TODO()
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
		return
	}
	testDBList := DatabaseList{
		databaseList: skiplist.NewSkipList[string, database.Database](),
		schema:       &testSchema,
	}
	testDB := database.Database{
		Name:      "testDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc1 := contents.Document{
		Name:    "testDoc1",
		Path:    "/v1/testDB/testDoc1",
		Content: []byte("{\"field1\":[]}"),
		Metadata: contents.Metadata{
			CreatedBy:      "julia",
			CreatedAt:      12,
			LastModifiedBy: "julia",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testDoc2 := contents.Document{
		Name:    "testDoc2",
		Path:    "/v1/testDB/testDoc2",
		Content: []byte("{\"field1\": { \"field2\":{}}}"),
		Metadata: contents.Metadata{
			CreatedBy:      "esther",
			CreatedAt:      12,
			LastModifiedBy: "esther",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testCollection1 := contents.Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	testDoc3 := contents.Document{
		Name:    "testDoc3",
		Path:    "/v1/testDB/testDoc2/testCollection1/testDoc3",
		Content: []byte("{\"field1\":[]}"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}
	testDoc100 := contents.Document{
		Name:    "testDoc100",
		Path:    "/v1/testDB/testDoc2/testCollection1/testDoc100",
		Content: []byte("{\"field1\": {\"field2\": {}}}"),
		Metadata: contents.Metadata{
			CreatedBy:      "april",
			CreatedAt:      12,
			LastModifiedBy: "april",
			LastModifiedAt: 13,
		},
		Collections: skiplist.NewSkipList[string, contents.Collection](),
		Subscribers: make(map[string]contents.WriteFlusher),
	}

	_, err := testDBList.databaseList.Upsert(testDB.Name, func(key string, currValue database.Database, exists bool) (database.Database, error) {
		return testDB, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDB: %v", err)
		return
	}
	_, err = testDB.Documents.Upsert(testDoc1.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
		return
	}
	_, err = testDB.Documents.Upsert(testDoc2.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
		return
	}
	_, err = testDoc2.Collections.Upsert(testCollection1.Name, func(key string, currValue contents.Collection, exists bool) (contents.Collection, error) {
		return testCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err)
		return
	}
	_, err = testCollection1.Documents.Upsert(testDoc3.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
		return
	}
	_, err = testCollection1.Documents.Upsert(testDoc100.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc100, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc100: %v", err)
		return
	}
	_, err1 := database.GetDatabase(testDBList.databaseList, "testDB", ctx, "", "")
	if err1 != nil {
		fmt.Printf("upserting the stuff in the test database failed")
		return
	} else {
		fmt.Print("ALL THE UPSERTING WORKED")
	}

	type PatchHandlerTest struct {
		r               *http.Request
		w               *httptest.ResponseRecorder
		expectedCode    int
		expectError     bool
		expectedDocBody []byte
		docName         string
		requestBody     []byte
		homeDB          database.Database
		homeCollection  contents.Collection
		topLevelDoc     bool
	}

	testCases := []PatchHandlerTest{
		//Test cases for top-level document in the database that should all work
		{
			// this case doesn't work, returns a bad uri response
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"ArrayAdd\", \"path\": \"/field1\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":[{\"field2\":5}]}"),
			docName:         "testDoc1",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc1\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			topLevelDoc:     true,
		},
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2", strings.NewReader("[{\"op\": \"ObjectAdd\", \"path\": \"/field1/field2\", \"value\": { \"key\": \"value\"}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":{\"field2\":{\"key\":\"value\"}}}"),
			docName:         "testDoc2",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc1\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			topLevelDoc:     true,
		},
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"ArrayRemove\", \"path\": \"/field1\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":[]}"),
			docName:         "testDoc1",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc1\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			//homeCollection: skiplist.NewSkipList[string, contents.Collection](),
			topLevelDoc: true,
		},
		//Test cases for nested document in the database that should all work
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2/testCollection1/testDoc3", strings.NewReader("[{\"op\": \"ArrayAdd\", \"path\": \"/field1\", \"value\": { \"nestedField\": \"yes\"}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":[{\"nestedField\":\"yes\"}]}"),
			docName:         "testDoc3",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc2/testCollection1/testDoc3\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			homeCollection:  testCollection1,
			topLevelDoc:     false,
		},
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2/testCollection1/testDoc100", strings.NewReader("[{\"op\": \"ObjectAdd\", \"path\": \"/field1/field2\", \"value\": { \"key2\": \"value2\"}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":{\"field2\":{\"key2\":\"value2\"}}}"),
			docName:         "testDoc100",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc1\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			homeCollection:  testCollection1,
			topLevelDoc:     false,
		},
		//Another remove bug
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2/testCollection1/testDoc3", strings.NewReader("[{\"op\": \"ArrayRemove\", \"path\": \"/field1\", \"value\": { \"nestedField\": \"yes\"}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     false,
			expectedDocBody: []byte("{\"field1\":[]}"),
			docName:         "testDoc3",
			requestBody:     []byte("{\"uri\": \"/v1/testDB/testDoc1\", \"patchFailed\": false, \"message\": \"patch applied\"}"),
			homeDB:          testDB,
			homeCollection:  testCollection1,
			topLevelDoc:     false,
		},

		// Test cases with valid bodies but non existing items in the path
		// Non existing DB
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB1/testDoc1", strings.NewReader("[{\"op\": \"ArrayAdd\", \"path\": \"/field1\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusNotFound,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc1",
			requestBody:     []byte(""),
			// These fields shouldn't be used
			// homeDB: testDB.Documents,
			// homeCollection: skiplist.NewSkipList[string, contents.Collection](),
			topLevelDoc: true,
		},
		// Non existing document
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc9/testCollection1/testDoc7", strings.NewReader("[{\"op\": \"ArrayRemove\", \"path\": \"/field1/field2\"}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusNotFound,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc7",
			// These fields shouldn't be used
			// homeDB: testDB.Documents,
			// homeCollection: skiplist.NewSkipList[string, contents.Collection](),
			topLevelDoc: true,
		},
		// Non existing nested document
		///WHAT IS HAPPENING HERE
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2/testCollection3/testDoc7", strings.NewReader("[{\"op\": \"ArrayRemove\", \"path\": \"/field1/field2\"}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusNotFound,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc7",
			// These fields shouldn't be used
			// homeDB: testDB.Documents,
			// homeCollection: skiplist.NewSkipList[string, contents.Collection](),
			topLevelDoc: true,
		},
		// Test cases for requests with bad path operations
		// {
		// 	r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"InvalidOp\", \"path\": \"/field1\", \"value\": { \"field2\": 5}}]")),
		// 	w:               httptest.NewRecorder(),
		// 	expectedCode:    http.StatusBadRequest,
		// 	expectError:     true,
		// 	expectedDocBody: []byte(""),
		// 	docName:         "testDoc1",
		// 	requestBody:     []byte(""),
		// 	// These fields shouldn't be used
		// 	// homeDB: testDB.Documents,
		// 	// homeCollection: skiplist.NewSkipList[string, contents.Collection](),
		// 	topLevelDoc: true,
		// },
		// {
		// 	r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc2/testCollection1/testDoc3", strings.NewReader("[{\"op\": \"IvalidObjectStuff\", \"path\": \"/field1/field2\", \"value\": { \"key\": \"value\"}}]")),
		// 	w:               httptest.NewRecorder(),
		// 	expectedCode:    http.StatusBadRequest,
		// 	expectError:     true,
		// 	expectedDocBody: []byte(""),
		// 	docName:         "testDoc3",
		// 	requestBody:     []byte(""),
		// 	// These fields shouldn't be used
		// 	// homeDB: testDB.Documents,
		// 	// homeCollection: skiplist.NewSkipList[string, contents.Collection](),
		// 	topLevelDoc: true,
		// },

		// Testing trying to add to a nonexistent field
		// these will throw an error but still return a StatusOK code
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"ArrayAdd\", \"path\": \"/field2\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc1",
			requestBody:     []byte(""),
			homeDB:          testDB,
			topLevelDoc:     true,
		},
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"ObjectAdd\", \"path\": \"/field2\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc1",
			requestBody:     []byte(""),
			homeDB:          testDB,
			topLevelDoc:     true,
		},
		{
			r:               httptest.NewRequest(http.MethodPatch, "/v1/testDB/testDoc1", strings.NewReader("[{\"op\": \"ArrayRemove\", \"path\": \"/field2\", \"value\": { \"field2\": 5}}]")),
			w:               httptest.NewRecorder(),
			expectedCode:    http.StatusOK,
			expectError:     true,
			expectedDocBody: []byte(""),
			docName:         "testDoc1",
			requestBody:     []byte(""),
			homeDB:          testDB,
			topLevelDoc:     true,
		},
	}
	i := 0
	for _, tc := range testCases {
		t.Run(tc.r.Method, func(t *testing.T) {

			testDBList.PatchHandler(tc.w, tc.r)
			fmt.Printf("expected CODE IS %d", tc.expectedCode)
			if tc.expectedCode != tc.w.Code {
				//errorMessage := fmt.Sprintf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				fmt.Printf("REQUEST BODY IS: %s", tc.r.Body)
				t.Fatalf("Expected codes do not match: expected %d and received %d", tc.expectedCode, tc.w.Code)
				return
			}
			if tc.expectError == false {
				if tc.w.Code == http.StatusMethodNotAllowed {
					//errorMessage := fmt.Sprintf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 405 was expected but received %d", tc.w.Code)
					return
				}
				var resultDoc contents.Document
				if tc.topLevelDoc == true {
					resultDoc, err = contents.GetDocument(tc.homeDB.Documents, tc.docName)
					if err != nil {
						t.Fatalf("Test case failed: could not retrieve document contents after patch was applied")
						return
					}
				} else {
					resultDoc, err = contents.GetDocument(tc.homeCollection.Documents, tc.docName)
					if err != nil {
						t.Fatalf("Test case failed: could not retrieve document contents after patch was applied")
						return
					}
				}
				for i := 0; i < len(resultDoc.Content); i++ {
					if resultDoc.Content[i] != tc.expectedDocBody[i] {
						t.Fatalf("Test case failed: expected contents do not match resulting contents.\n Expected: %s\n, Resulting: %s", string(tc.expectedDocBody), string(resultDoc.Content))
						return
					}
				}
			} else {
				if tc.w.Code == http.StatusOK && i < len(testCases)-4 {
					//errorMessage := fmt.Sprintf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					t.Fatalf("Test failed, status code 400 was expected but received %d", tc.w.Code)
					return
				}
			}
		})
		i += 1
	}
}
