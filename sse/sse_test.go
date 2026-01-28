package sse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
)

// fake Subscriber factory used for tests
func SubscriberFactoryforTest() DBIndex[string, *Subscriber] {
	return skiplist.NewSkipList[string, *Subscriber]()
}

// Struct used for our test cases for SubScribePath
type testSubPath struct {
	resource    string
	expectError bool
	expectedSub bool
}

// Test function for SubScribePath function in SSE
func TestSubscribePath(t *testing.T) {
	// Making the fake subscribe habdler
	resourceToken := skiplist.NewSkipList[string, DBIndex[string, *Subscriber]]()
	testSubHandler := NewSubscriberHandler(resourceToken, SubscriberFactoryforTest)

	// test cases
	testCases := []testSubPath{
		// these should all work
		{
			resource:    "julia",
			expectError: false,
			expectedSub: true,
		},
		{
			resource:    "april",
			expectError: false,
			expectedSub: true,
		},
		{
			resource:    "esther",
			expectError: false,
			expectedSub: true,
		},
		// this won't work because it has already been added
		{
			resource:    "esther",
			expectError: false,
			expectedSub: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.resource, func(t *testing.T) {
			err := testSubHandler.SubscribePath(tc.resource)
			if err != nil {
				// checking that an unexpected error didn't occur
				if tc.expectError == false {
					t.Fatalf(" Test case failed: did not expect an error but one was returned.")
					return
				}
			} else {
				// checking if an unexpected error occurred
				if tc.expectedSub == true {
					_, found := resourceToken.Find(tc.resource)
					if found == false {
						t.Fatalf("Test case failed, resource couldn't be found in the skiplist even though it was supposed to be added")
						return
					}
				}
			}
		})
	}
}

// test struct used for testing AddSubscription
type testAddSub struct {
	r           *http.Request
	resource    string
	token       string
	expectError bool
}

// Test function for AddSubscription for SSE
func TestAddSubScription(t *testing.T) {
	// creating the fake resource token and subscription handler
	resourceToken := skiplist.NewSkipList[string, DBIndex[string, *Subscriber]]()
	testSubHandler := NewSubscriberHandler(resourceToken, SubscriberFactoryforTest)

	resource1 := "path1"
	resource2 := "path1/path2"

	// upserting the fake elements
	_, err := testSubHandler.resourceToken.Upsert(resource1, func(key string, currValue DBIndex[string, *Subscriber], exists bool) (DBIndex[string, *Subscriber], error) {
		if exists {
			return currValue, nil
		}
		return SubscriberFactoryforTest(), nil
	})
	if err != nil {
		t.Fatalf("Failed to Upsert %s", resource1)
	}
	_, err = testSubHandler.resourceToken.Upsert(resource2, func(key string, currValue DBIndex[string, *Subscriber], exists bool) (DBIndex[string, *Subscriber], error) {
		if exists {
			return currValue, nil
		}
		return SubscriberFactoryforTest(), nil
	})
	if err != nil {
		t.Fatalf("Failed to Upsert %s", resource2)
	}

	testCases := []testAddSub{
		//first two should work
		{
			r:           httptest.NewRequest(http.MethodGet, "/"+resource1, nil),
			resource:    resource1,
			token:       "madeupstring",
			expectError: false,
		},
		{
			r:           httptest.NewRequest(http.MethodGet, "/"+resource2, nil),
			resource:    resource2,
			token:       "madeupstringnumber2",
			expectError: false,
		},
		// the test below won't work because it has already been added
		{
			r:           httptest.NewRequest(http.MethodGet, "/"+resource2, nil),
			resource:    resource2,
			token:       "madeupstringnumber2",
			expectError: true,
		},
		// this test won't work because it is a non existing resource
		{
			r:           httptest.NewRequest(http.MethodGet, "/resource", nil),
			resource:    "resource3",
			token:       "this is a fake resource",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		// calling the function
		err = testSubHandler.addSubscription(tc.resource, tc.token, tc.r)
		// checking that an error was expected if it occured
		if err != nil {
			if tc.expectError == false {
				t.Fatalf("Test case failed, an error was thrown when one was not expected")
				return
			}
		} else {
			// checking that an error was not missed
			if tc.expectError == true {
				t.Fatalf("Test case failed: an error was not return when one was expected")
				return
			}
		}
	}
}

// type testDeleteSub struct {
// 	resource    string
// 	token       string
// 	expectError bool
// }

// func TestDeleteSubscription(t *testing.T) {
// 	resourceToken := skiplist.NewSkipList[string, DBIndex[string, *Subscriber]]()
// 	testSubHandler := NewSubscriberHandler(resourceToken, SubscriberFactoryforTest)

// 	resource1 := "path1"
// 	resource2 := "path1/path2"

// 	_, err := testSubHandler.resourceToken.Upsert(resource1, func(key string, currValue DBIndex[string, *Subscriber], exists bool) (DBIndex[string, *Subscriber], error) {
// 		if exists {
// 			return currValue, nil
// 		}
// 		return SubscriberFactoryforTest(), nil
// 	})
// 	if err != nil {
// 		t.Fatalf("Failed to Upsert %s", resource1)
// 	}
// 	_, err = testSubHandler.resourceToken.Upsert(resource2, func(key string, currValue DBIndex[string, *Subscriber], exists bool) (DBIndex[string, *Subscriber], error) {
// 		if exists {
// 			return currValue, nil
// 		}
// 		return SubscriberFactoryforTest(), nil
// 	})
// 	if err != nil {
// 		t.Fatalf("Failed to Upsert %s", resource2)
// 	}

// 	testCases := []testDeleteSub{
// 		{
// 			resource:    resource1,
// 			token:       "token1",
// 			expectError: false,
// 		},
// 		{
// 			resource:    resource2,
// 			token:       "token2",
// 			expectError: false,
// 		},
// 		{
// 			resource:    "resource3",
// 			token:       "token3",
// 			expectError: true,
// 		},
// 	}
// 	for _, tc := range testCases {
// 		err1 := testSubHandler.deleteSubscription(tc.resource, tc.token)
// 		if err1 != nil {
// 			if tc.expectError == false {
// 				t.Fatalf("Test case failed: an error was returned when one was not expected")
// 				return
// 			}
// 		} else {
// 			if tc.expectError == true {
// 				t.Fatalf("Test case failed: an error should have been returned but no errors were returned")
// 				return
// 			}
// 		}
// 	}
// }
