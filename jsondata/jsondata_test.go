package jsondata

import (
	"fmt"
	"testing"
)

type test struct {
	jsonfile    string
	expectError bool
	errMessage  string
}

func TestNewSchema(t *testing.T) {
	testCases := []test{
		{
			jsonfile:    "../schema1.json",
			expectError: false,
			errMessage:  "",
		},
		{
			jsonfile:    "../schema2.json",
			expectError: false,
		},
		{
			jsonfile:    "../schema3.json",
			expectError: false,
			errMessage:  "",
		},
		{
			jsonfile:    "../schema4.json",
			expectError: false,
			errMessage:  "",
		},
		{
			jsonfile:    "../schema5.json",
			expectError: true,
			errMessage:  "Schema file does not exist",
		},
		{
			jsonfile:    "../notASchema.json",
			expectError: true,
			errMessage:  "Schema file does not contain a valid schema",
		},
		{
			jsonfile:    "",
			expectError: true,
			errMessage:  "No filename for schema was provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.jsonfile, func(t *testing.T) {
			_, err := New(tc.jsonfile)

			if tc.expectError == true {
				if err != nil {
					return
				}
			}
			if err != nil && tc.expectError == false {
				t.Fatalf("Unexpected error when one should not have occured")
			}
		})
	}
}

type validateTest struct {
	documentContent string
	schema          ValidSchema
	expectError     bool
	errMessage      string
}

func TestValidateSchema(t *testing.T) {
	schema1, err := New("../schema1.json")
	if err != nil {
		t.Fatalf("ERROR: Test Schema 1 did not compile properly")
	}
	schema2, err := New("../schema2.json")
	if err != nil {
		t.Fatalf("ERROR: Test Schema 1 did not compile properly")
	}

	testCases := []validateTest{
		{
			documentContent: `{"name": "Julia", "age": 22}`,
			schema:          schema1,
			expectError:     false,
			errMessage:      "",
		},
		{
			documentContent: `{"name": "esther", "age": 24}`,
			schema:          schema1,
			expectError:     false,
			errMessage:      "",
		},
		{
			documentContent: `{"name": "esther", "age": "20"}`,
			schema:          schema1,
			expectError:     true,
			errMessage:      "Content does not match schema: string was provided instead of integer for 'age' field",
		},
		{
			documentContent: `{"age": "20"}`,
			schema:          schema1,
			expectError:     true,
			errMessage:      "Content does not match schema: name field is not present",
		},
		{
			documentContent: `{"name": "esther", "age": "20"}`,
			schema:          schema2,
			expectError:     true,
			errMessage:      "Does not match schema at all",
		},
		{
			documentContent: ` `,
			schema:          schema1,
			expectError:     true,
			errMessage:      "",
		},
		{
			documentContent: `blah blah blah`,
			schema:          schema1,
			expectError:     true,
			errMessage:      "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.documentContent, func(t *testing.T) {
			results, err := tc.schema.ValidateDocument([]byte(tc.documentContent))
			if tc.expectError == true {
				if results != false {
					fmt.Printf("Expected error was: %s", tc.errMessage)
					t.Fatal("Expected error for but received none")
				}
				return
			}

			if err != nil {
				t.Fatalf("An error occured when one was not expected")
			}
		})
	}
}
