package contents

import (
	"context"
	"fmt"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/jsondata"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
)

type collectionTest struct {
	name              string
	start             string
	end               string
	expectedNumOfDocs int
	expectedRes       Collection
	expectError       bool
}

func TestGetCollection(t *testing.T) {
	ctx := context.TODO()
	testCollectionList := skiplist.NewSkipList[string, Collection]()

	testCollection1 := Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, Document](),
	}
	testCollection3 := Collection{
		Name:      "testCollection3",
		Documents: skiplist.NewSkipList[string, Document](),
	}
	testDoc1 := Document{
		Name:        "testDoc1",
		Collections: skiplist.NewSkipList[string, Collection](),
	}
	testDoc2 := Document{
		Name:        "testDoc2",
		Collections: skiplist.NewSkipList[string, Collection](),
	}
	testDoc3 := Document{
		Name:        "testDoc3",
		Collections: skiplist.NewSkipList[string, Collection](),
	}
	testDoc4 := Document{
		Name:        "testDoc4",
		Collections: skiplist.NewSkipList[string, Collection](),
	}

	_, err1 := testCollectionList.Upsert(testCollection1.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testCollection1, nil
	})
	if err1 != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err1)
	}
	_, err := testCollectionList.Upsert(testCollection3.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testCollection3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert collection3: %v", err)
	}
	_, err = testCollection1.Documents.Upsert(testDoc1.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1")
	}
	_, err = testCollection1.Documents.Upsert(testDoc2.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc2")
	}
	_, err = testCollection1.Documents.Upsert(testDoc3.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3")
	}
	_, err = testCollection1.Documents.Upsert(testDoc4.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc4")
	}

	testCases := []collectionTest{
		{
			name:              "testCollection1",
			start:             "testDoc1",
			end:               "testDoc4",
			expectedRes:       testCollection1,
			expectedNumOfDocs: 4,
			expectError:       false,
		},
		{
			name:              "collection2",
			start:             "testDoc1",
			end:               "testDoc4",
			expectedNumOfDocs: 0,
			expectError:       true,
		},
		{
			name:              " ",
			start:             "testDoc1",
			end:               "testDoc4",
			expectedNumOfDocs: 0,
			expectError:       true,
		},
		{
			name:              "testCollection3",
			start:             "",
			end:               "",
			expectedRes:       testCollection3,
			expectedNumOfDocs: 0,
			expectError:       false,
		},
		{
			name:              "testCollection3",
			start:             "lol1",
			end:               "zzz",
			expectedRes:       testCollection3,
			expectedNumOfDocs: 0,
			expectError:       false,
		},
		{
			name:              "testCollection1",
			start:             "testDoc2",
			end:               "testDoc4",
			expectedRes:       testCollection1,
			expectedNumOfDocs: 3,
			expectError:       false,
		},
		{
			name:              "testCollection1",
			start:             "testDoc3",
			end:               "testDoc5",
			expectedRes:       testCollection1,
			expectedNumOfDocs: 4,
			expectError:       false,
		},
		{
			name:              "testCollection1",
			start:             "testDoc0",
			end:               "testDoc3",
			expectedRes:       testCollection1,
			expectedNumOfDocs: 3,
			expectError:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			col, err := GetCollection(testCollectionList, tc.name, ctx, "", "")

			// Check if there was an expected error that was missed
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error but received none")
					return
				}
				return
			}

			// Check for an unexpected error
			if err != nil {
				t.Fatalf("Did not expect an error but got one")
			}

			// Verify the collection returned
			if col.Name != tc.expectedRes.Name {
				t.Fatalf("Expected collection %v but got %v", tc.expectedRes.Name, col.Name)
			}
		})
	}
}

type putCollectionTest struct {
	name          string
	expectedError bool
}

func TestPutCollection(t *testing.T) {
	ctx := context.TODO()
	testCollectionList := skiplist.NewSkipList[string, Collection]()

	testCases := []putCollectionTest{
		{
			name:          "collection1",
			expectedError: false,
		},
		{
			name:          "collection2",
			expectedError: false,
		},
		{
			name:          "collection1",
			expectedError: true,
		},
		{
			name:          " ",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := PutCollection(testCollectionList, tc.name)

			// Checking for expected errors
			if tc.expectedError {
				if results {
					t.Fatalf("Put was successful but returned an error")
				}
				if err == nil {
					t.Fatalf("Expected an error but nothing was returned")
				}
				return
			}

			// Checking for unexpected errors
			if err != nil {
				t.Fatalf("Error received when one was not expected: %v", err)
			}
			if !results {
				t.Fatalf("Test failed but no error was returned")
			}
			coll, err := GetCollection(testCollectionList, tc.name, ctx, "", "")
			if err != nil {
				t.Fatalf("Test case failed: collection could not be retrieved after a 'successful' Put")
				return
			}

			if coll.Name != tc.name {
				t.Fatalf("Test case failed: resulting collection's name did not match the test case's name")
				return
			}
		})
	}

	// Check if collections exist
	collections, err := testCollectionList.Query(ctx, "", "")
	if err != nil {
		t.Fatalf("Collections should exist but were not found: %v", err)
	}
	if len(collections) != 3 {
		t.Fatalf("Expected %v collections but got %v", 3, len(collections))
	}
}

type deleteCollectionTest struct {
	name          string
	expectedError bool
}

func TestDeleteCollection(t *testing.T) {
	ctx := context.TODO()
	testCollectionList := skiplist.NewSkipList[string, Collection]()

	testCollection1 := Collection{
		Name:      "testCollection1",
		Documents: skiplist.NewSkipList[string, Document](),
	}
	testCollection2 := Collection{
		Name:      "testCollection2",
		Documents: skiplist.NewSkipList[string, Document](),
	}
	testDoc1 := Document{
		Name:        "testDoc1",
		Collections: skiplist.NewSkipList[string, Collection](),
	}
	testDoc2 := Document{
		Name:        "testDoc2",
		Collections: skiplist.NewSkipList[string, Collection](),
	}

	_, err := testCollectionList.Upsert(testCollection1.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testCollection1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection1: %v", err)
	}
	_, err = testCollectionList.Upsert(testCollection2.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testCollection2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testCollection2: %v", err)
	}
	_, err = testCollection1.Documents.Upsert(testDoc1.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testCollection1.Documents.Upsert(testDoc2.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc2: %v", err)
	}

	testCases := []deleteCollectionTest{
		{
			name:          "testCollection1",
			expectedError: false,
		},
		{
			name:          "testCollection2",
			expectedError: false,
		},
		{
			name:          "collection3",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := DeleteCollection(testCollectionList, tc.name)

			// Checking for expected errors
			if tc.expectedError {
				if results {
					t.Fatalf("Delete was successful but returned an error")
				}
				if err == nil {
					t.Fatalf("Expected an error but nothing was returned")
				}
				return
			}

			// Checking for unexpected errors
			if err != nil {
				t.Fatalf("Error received when one was not expected: %v", err)
			}
			if !results {
				t.Fatalf("Test failed but no error was returned")
			}
		})
	}

	// Check remaining collections
	collections, err := testCollectionList.Query(ctx, "", "")
	if err != nil {
		t.Fatalf("Collections should exist but were not found: %v", err)
	}
	if len(collections) != 0 {
		t.Fatalf("Expected %v collections but got %v", 0, len(collections))
	}
}

type testGetDoc struct {
	docName      string
	docContents  []byte
	expectError  bool
	errMessage   string
	expectedBody []byte
}

func TestGetDocument(t *testing.T) {

	testCollection := skiplist.NewSkipList[string, Document]()
	testDoc1 := Document{Name: "testDoc1", Content: []byte("this is a test Document")}
	testDoc2 := Document{Name: "testDoc2", Content: []byte("this is a another test Document")}
	testDoc3 := Document{Name: "testDoc3", Content: []byte("")}

	_, err := testCollection.Upsert(testDoc1.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testCollection.Upsert(testDoc2.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc2: %v", err)
	}
	_, err = testCollection.Upsert(testDoc3.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
	}

	testCases := []testGetDoc{
		{
			docName:      "testDoc1",
			docContents:  testDoc1.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte("this is a test Document"),
		},
		{
			docName:      "testDoc2",
			docContents:  testDoc2.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte("this is a another test Document"),
		},
		{
			docName:      "testDoc3",
			docContents:  testDoc3.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte(""),
		},
		{
			docName:      "testDoc4",
			docContents:  []byte("body for a document that does not exist"),
			expectError:  true,
			errMessage:   "",
			expectedBody: []byte(""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.docName, func(t *testing.T) {
			results, err := GetDocument(testCollection, tc.docName)
			if tc.expectError == true {
				if err == nil {
					fmt.Printf("Expected error message was: %s", tc.errMessage)
					t.Fatalf("Test failed: error was not return when expected")
					return
				}
			} else if err != nil {
				t.Fatalf("Unexpected error occured, err: %s", tc.errMessage)
				return
			} else {
				if tc.docName != results.Name {
					t.Fatalf("Document Name does not match expected result: expected %s and got %s", tc.docName, results.Name)
					return
				}

				if len(tc.docContents) != len(results.Content) {
					fmt.Printf("Expected body length: %d, Result body length: %d", len(tc.docContents), len(results.Content))
					t.Fatalf("Length of Result contents does not match the length of the expected body")
					return
				}
				for i := 0; i < len(tc.docContents); i++ {
					if tc.docContents[i] != results.Content[i] {
						fmt.Printf("Result content: %b, Expected content: %b", results.Content[i], tc.docContents[i])
						t.Fatalf("Result contents do not match expected body contents")
						return
					}
				}
			}

		})
	}
}

type testPutDoc struct {
	docName     string
	docContents []byte
	user        string
	mode        string
	schema      jsondata.ValidSchema
	expectError bool
	errMessage  string
}

func TestPutDocument(t *testing.T) {
	ctx := context.TODO()
	anySchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
		return
	}
	nameAgeSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
		return
	}
	testCollection := skiplist.NewSkipList[string, Document]()

	testCases := []testPutDoc{

		{
			docName:     "test1Doc1",
			docContents: []byte("{\"julia\":2}"),
			user:        "april",
			mode:        "",
			schema:      anySchema,
			expectError: false,
			errMessage:  "Test Case 1: this test case should pass",
		},
		{
			docName:     "testDoc2",
			docContents: []byte("{\"beeboop\":2}"),
			user:        "esther",
			mode:        "",
			schema:      anySchema,
			expectError: false,
			errMessage:  "Test Case 2: This also should pass",
		},
		{
			docName:     "test1Doc1",
			docContents: []byte("{\"i am\": tired}"),
			user:        "jim",
			mode:        "nooverwrite",
			schema:      anySchema,
			expectError: true,
			errMessage:  "Test Case 3: This should fail because we are trying to overwrite an existing document in noOverwrite mode",
		},
		{
			docName:     "test4Doc4",
			docContents: []byte("{\"name\": \"julia\", \"age\": 12}"),
			user:        "jim",
			mode:        "nooverwrite",
			schema:      nameAgeSchema,
			expectError: false,
			errMessage:  "Test Case 4: This should pass because we are matching the content with a given schema",
		},
		{
			docName:     "test5Doc5",
			docContents: []byte(""),
			user:        "jim",
			mode:        "",
			schema:      nameAgeSchema,
			expectError: true,
			errMessage:  "Test Case 5: This should fail because the contents of the test case do not match the schema",
		},
		{
			docName:     "test6Doc6",
			docContents: []byte("{\"comp318\": \"fun!\"}"),
			user:        "esther",
			mode:        "",
			schema:      anySchema,
			expectError: false,
			errMessage:  "Test case 6: This should pass",
		},
		{
			docName:     "test1Doc1",
			docContents: []byte("{\"hello\": \"world!\"}"),
			user:        "esther",
			mode:        "overwrite",
			schema:      anySchema,
			expectError: false,
			errMessage:  "Test case 7: This should pass and should overwrite the previous contents",
		},
		{
			docName:     "test6Doc6",
			docContents: []byte("{\"overwrite\": \"testing!\"}"),
			user:        "julia",
			mode:        "",
			schema:      anySchema,
			expectError: false,
			errMessage:  "Test case 6: This should pass",
		},
	}
	testSchema, err2 := jsondata.New("../schemaAny.json")
	if err2 != nil {
		t.Fatalf("Test schema could not be successfully created")
	}

	for _, tc := range testCases {
		t.Run(tc.docName, func(t *testing.T) {
			results, err := PutDocument(testCollection, tc.docName, tc.docContents, tc.user, tc.mode, &testSchema)
			if tc.expectError == true {
				fmt.Printf("Doc1 Name %s", tc.docName)
				if results == true || err == nil {
					fmt.Printf("Expected error message: %s", tc.errMessage)
					t.Fatalf("Test case failed: an error was not returned when one was expected")
					return
				}
			} else {
				if err != nil {
					if results == false {
						fmt.Printf("TEM: %s", tc.errMessage)
						fmt.Printf("Test case failed when it should have passed")
						t.Fatalf("TEM: %s, ERROR MES: %s, RESULTS:", tc.errMessage, err)
						return
					} else {
						t.Fatalf("Test case failed but the put function reported success")
						return
					}
				}
				checkRes, err := GetDocument(testCollection, tc.docName)
				if err != nil {
					t.Fatalf("Test case failed: a document was successfully put but could not be retrieved after")
					return
				}
				if checkRes.Name != tc.docName {
					t.Fatalf("Test case failed: result document's name does not match the test case's name")
					return
				}
			}
		})
	}

	results, err1 := testCollection.Query(ctx, "", "")
	if err1 != nil {
		t.Fatalf("Error with retrieving test documents")
		return
	}
	if len(results) != 4 {
		t.Fatalf("Test cases failed, expected 4 document as a result but %d are in the database", len(results))
	}
	for _, document := range results {
		if document.Name == "test1Doc1" {
			if document.Metadata.LastModifiedBy != "esther" {
				if document.Metadata.LastModifiedBy == "jim" {
					t.Fatalf("Overwrite functionality is incorrect, document did not properly ovewrite existing content")
					return
				} else {
					fmt.Printf("LAST USER WAS %s", document.Metadata.LastModifiedBy)
					t.Fatalf("NoOverwrite functionality is incorrect, metadata was written over when it should not have been")
					return
				}
			}
		} else if document.Name == "test6Doc6" {
			if document.Metadata.LastModifiedBy != "julia" {
				fmt.Printf("LAST USER WAS %s", document.Metadata.LastModifiedBy)
				t.Fatalf("NoOverwrite functionality is incorrect, metadata was written over when it should not have been")
				return
			}
		}
	}
}

type testDeleteDoc struct {
	docName string
	// docContents  []byte
	expectError  bool
	errMessage   string
	expectedBody []byte
}

func TestDeleteDocument(t *testing.T) {

	testCollection := skiplist.NewSkipList[string, Document]()
	testDoc1 := Document{Name: "testDoc1", Collections: skiplist.NewSkipList[string, Collection]()}
	testDoc2 := Document{Name: "testDoc2", Collections: skiplist.NewSkipList[string, Collection]()}
	testDoc3 := Document{Name: "testDoc3", Collections: skiplist.NewSkipList[string, Collection]()}

	testNestedCol1 := Collection{Name: "testNestedCol1", Documents: skiplist.NewSkipList[string, Document]()}
	testNestedCol2 := Collection{Name: "testNestedCol2", Documents: skiplist.NewSkipList[string, Document]()}
	testNestedCol3 := Collection{Name: "testNestedCol3", Documents: skiplist.NewSkipList[string, Document]()}

	_, err := testCollection.Upsert(testDoc1.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc1: %v", err)
	}
	_, err = testCollection.Upsert(testDoc2.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc2: %v", err)
	}
	_, err = testCollection.Upsert(testDoc3.Name, func(key string, currValue Document, exists bool) (Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
	}
	_, err = testDoc1.Collections.Upsert(testNestedCol1.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testNestedCol1, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testNestedCol1: %v", err)
	}
	_, err = testDoc1.Collections.Upsert(testNestedCol2.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testNestedCol2, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testNestedCol2: %v", err)
	}
	_, err = testDoc2.Collections.Upsert(testNestedCol3.Name, func(key string, currValue Collection, exists bool) (Collection, error) {
		return testNestedCol3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testNestedCol3: %v", err)
		return
	}

	fmt.Printf("IT WORKED UPSERT")
	testCases := []testDeleteDoc{
		{
			docName: "testDoc1",
			// docContents:  testDoc1.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte("this is a test Document"),
		},
		{
			docName: "testDoc2",
			// docContents: testDoc2.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte("this is a another test Document"),
		},
		{
			docName: "testDoc3",
			// docContents: testDoc3.Content,
			expectError:  false,
			errMessage:   "",
			expectedBody: []byte(""),
		},
		// {
		// 	docName: "testDoc4",
		// 	// docContents: []byte("body for a document that does not exist"),
		// 	expectError:  true,
		// 	errMessage:   "",
		// 	expectedBody: []byte(""),
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.docName, func(t *testing.T) {
			results, err := DeleteDocument(testCollection, tc.docName)
			if tc.expectError == true {
				if err == nil {
					fmt.Printf("Expected error message was: %s", tc.errMessage)
					t.Fatalf("Test failed: error was not return when expected")
					return
				} else if results == true {
					t.Fatalf("Test failed: delete operation was reported successful when an error was expected")
					return
				}
			} else if err != nil {
				t.Fatalf("Unexpected error occured, err: %s", tc.errMessage)
				return
			}
			fmt.Printf("GOT THRU TEST")
		})
	}
}
