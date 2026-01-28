// Package jsodata contains a struct that holds the valid schema provided
// at the initialization of the server. It also contains functions that will
// compile the provided JSON Schema and validate a document's contents with
// the schema.

package jsondata

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// This struct is used to store the JSON Schema that is provided when starting the server.
// It is used to validate the contents of a new document upon its creation.
type ValidSchema struct {
	schema *jsonschema.Schema
}

// New initializes a new ValidSchema struct, takes a JSON Schema file as input,
// compiles the given file, and then stores the schema in the ValidSchema struct.
func New(jsonflag string) (ValidSchema, error) {

	compiler := jsonschema.NewCompiler()

	// Compiling the JSON Schema
	sch, err := compiler.Compile(jsonflag)

	// Checking that compilation was successful
	if err != nil {
		fmt.Printf("Unable to compile provided schema\n")
		return ValidSchema{schema: nil}, err
	}
	// Storing the schema in a ValidSchema struct
	newSchema := ValidSchema{schema: sch}

	return newSchema, nil
}

// ValidateDocument is used in the contents package when creating new documents. This function
// takes a valid schema and the content of a document as input. This function will call the Validate()
// method to ensure that the document content matches the provided schema.
//
// If the provided schema is empty, the document content is unable to be unmarshalled, or
// the content cannot be validated with the schema, then the function will return false and a corresponding
// error message.
// If the validation succeeds, the function will return true with no error (nil).
func (sch *ValidSchema) ValidateDocument(documentContent []byte) (bool, error) {
	schema := sch.schema
	if schema == nil {
		return false, nil
	}
	var unmarshalled any
	if err := json.Unmarshal(documentContent, &unmarshalled); err != nil {
		return false, err
	}

	if err := schema.Validate(unmarshalled); err != nil {
		fmt.Printf("document content does not conform to the provided schema\n")
		return false, err
	}
	return true, nil
}
