package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/contents"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/sse"
)

type test struct {
	path        string
	start       string
	end         string
	expectedRes int
	expectError bool
}

func TestGetDatabase(t *testing.T) {
	ctx := context.TODO()

	testDBList := skiplist.NewSkipList[string, Database]()

	testDB := Database{
		Name:      "testDB",
		Documents: skiplist.NewSkipList[string, contents.Document](),
	}
	if testDB.Documents == nil {
		t.Fatalf("nil pointer here")
	}
	testDoc1 := contents.Document{Name: "testDoc1"}
	testDoc2 := contents.Document{Name: "testDoc2"}
	testDoc3 := contents.Document{Name: "testDoc3"}

	_, err := testDBList.Upsert(testDB.Name, func(key string, currValue Database, exists bool) (Database, error) {
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
		t.Fatalf("Failed to upsert testDoc2: %v", err)
	}
	_, err = testDB.Documents.Upsert(testDoc3.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
		return testDoc3, nil
	})
	if err != nil {
		t.Fatalf("Failed to upsert testDoc3: %v", err)
	}

	testCases := []test{
		// Test case: query the whole database
		{
			path:        "testDB",
			start:       "",
			end:         "",
			expectedRes: 3,
			expectError: false,
		},
		// Test case: query a range within the database
		{
			path:        "testDB",
			start:       "testDoc1",
			end:         "testDoc2",
			expectedRes: 2,
			expectError: false,
		},
		// Test case: query an existent range with a low and no high
		{
			path:        "testDB",
			start:       "testDoc1",
			end:         "",
			expectedRes: 3,
			expectError: false,
		},
		// Test case: query an existent range with no low and a high
		{
			path:        "testDB",
			start:       "",
			end:         "testDoc3",
			expectedRes: 3,
			expectError: false,
		},
		// Test case: querying a non-existing database
		{
			path:        "nonExistingDB",
			start:       "",
			end:         "",
			expectedRes: 0,
			expectError: true,
		},
		// Test case: querying a range of documents with the high being nonexistent
		{
			path:        "testDB",
			start:       "testDoc2",
			end:         "testDoc4",
			expectedRes: 2,
			expectError: false,
		},
		// Test case: querying a range of documents with the high being nonexistent
		{
			path:        "testDB",
			start:       "testDoc0",
			end:         "testDoc3",
			expectedRes: 3,
			expectError: false,
		},
		// Test case: querying a range of existing documents with the high being before the low
		{
			path:        "testDB",
			start:       "testDoc3",
			end:         "testDoc1",
			expectedRes: 0,
			expectError: false,
		},
	}

	// GO through test cases
	i := 0
	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			db, err := GetDatabase(testDBList, tc.path, ctx, tc.start, tc.end)
			dbContents := db.Documents
			// Check if there was an expected error that was missed
			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error but received none")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Did not expect an error but got: %v", err)
				}

				// Verify the documents returned in the database
				var actualResult []string
				resultDocs, err := dbContents.Query(ctx, tc.start, tc.end)
				if err != nil {
					t.Fatalf("Unexpected error when querying documents in the test database")
				}

				for _, docName := range resultDocs {
					actualResult = append(actualResult, docName.Name)
				}

				// Check if the result matches the expected result
				if len(actualResult) != tc.expectedRes {
					t.Fatalf("Expected %d documents but got %d for test case %d", tc.expectedRes, len(actualResult), i)
				}
			}
		})
		i += 1
	}
}

type putTest struct {
	path          string
	expectedError bool
}

func TestPutDatabase(t *testing.T) {
	fmt.Printf("testing put")
	ctx := context.TODO()

	testDBList := skiplist.NewSkipList[string, Database]()
	path := "/v1/db/doc"

	// Create SubscriberHandler instance
	subscriptionFactory := func() sse.DBIndex[string, *sse.Subscriber] {
		return skiplist.NewSkipList[string, *sse.Subscriber]()
	}
	resourceToken := skiplist.NewSkipList[string, sse.DBIndex[string, *sse.Subscriber]]()
	subscriberHandler := sse.NewSubscriberHandler(resourceToken, subscriptionFactory)

	testCases := []putTest{
		{
			path:          "testDB",
			expectedError: false,
		},
		{
			path:          "h",
			expectedError: false,
		},
		{
			// WE need an error for pre-existing databases, 400
			path:          "testDB",
			expectedError: true,
		},
		{
			// WE need an error for pre-existing databases, 400
			path:          " ",
			expectedError: false,
		},
		{
			// WE need an error for pre-existing databases, 400
			path:          "	",
			expectedError: false,
		},
		// {
		// 	// needs to return a 400 not 404
		// 	path:          "test/DB",
		// 	expectedError: true,
		// },
		// {
		// 	// 400
		// 	path:          "/testDB",
		// 	expectedError: true,
		// },
		// {
		// 	// 400
		// 	path:          "testDB/",
		// 	expectedError: true,
		// },
		// {
		// 	// 201
		// 	path:          "test DB",
		// 	expectedError: true,
		// },
		// {
		// 	// 201
		// 	path:          " ",
		// 	expectedError: true,
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			results, err := PutDatabase(testDBList, tc.path, subscriberHandler, path)
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
				t.Fatalf("Error received when one was not expected")
			}
			if !results {
				t.Fatalf("Test failed but no error was returned")
			}
		})
	}
	databases, err := testDBList.Query(ctx, "", "")
	if err != nil {
		t.Fatalf("Database testDB should exist but was not found: %v", err)
	}
	if len(databases) != 4 {
		t.Fatalf("Expected %v documents but got %v", 4, len(databases))
	}
}

type deleteTest struct {
	database      Database
	expectedError bool
}

func TestDeleteDatabase(t *testing.T) {
	ctx := context.TODO()
	path := "/v1/db/doc"
	testDBList := skiplist.NewSkipList[string, Database]()
	// Create SubscriberHandler instance
	subscriptionFactory := func() sse.DBIndex[string, *sse.Subscriber] {
		return skiplist.NewSkipList[string, *sse.Subscriber]()
	}
	resourceToken := skiplist.NewSkipList[string, sse.DBIndex[string, *sse.Subscriber]]()
	subscriberHandler := sse.NewSubscriberHandler(resourceToken, subscriptionFactory)

	PutDatabase(testDBList, "db1", subscriberHandler, path)
	PutDatabase(testDBList, "db2", subscriberHandler, path)
	PutDatabase(testDBList, "db3", subscriberHandler, path)

	testCases := []deleteTest{
		{
			database: Database{
				Name:      "db1",
				Documents: skiplist.NewSkipList[string, contents.Document](),
			},
			expectedError: false,
		},
		{
			database: Database{
				Name:      "db2",
				Documents: skiplist.NewSkipList[string, contents.Document](),
			},
			expectedError: false,
		},
		{
			database: Database{
				Name:      "db4",
				Documents: skiplist.NewSkipList[string, contents.Document](),
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.database.Name, func(t *testing.T) {
			results, err := DeleteDatabase(testDBList, tc.database, subscriberHandler, path)
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
				t.Fatalf("Error received when one was not expected")
			}
			if !results {
				t.Fatalf("Test failed but no error was returned")
			}
		})
	}

	databases, err := testDBList.Query(ctx, "", "")
	if err != nil {
		t.Fatalf("Database testDB should exist but was not found: %v", err)
	}
	for _, database := range databases {
		if database.Name != "db3" {
			t.Fatalf("Wrong databases were deleted")
		}
		// fmt.Printf("DATABASES ARE")
		// fmt.Printf(name)
	}
	if len(databases) != 1 {
		t.Fatalf("Expected %v documents but got %v", 1, len(databases))
	}

}