// Package database implements the structs used to represent databases.
// It also implements the functions needed to GET, PUT, and DELETE databases.
// These functions will be used in their respective handler functions.

package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/contents"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/sse"
)

// This Database struct is how we represent databases. The struct holds the
// name of the database and a list of its top-level documents.
type Database struct {
	Name      string
	Documents skiplist.DBIndex[string, contents.Document] // initialized as an empty map
}

// GetDatabase retrieves all (or some) contents of a database in the given databaseList with the name databaseName.
// If a database with the given name could not be found, then the function returns an empty database with
// an error.
//
// GetDatabase can either retrieve all of the contents in a database, or will only retrieve contents within a
// range of two names of documents. The 'start' and 'end' points are represented as strings. If the input values for 'start'
// and 'end' are empty strings, then GetDatabase will return the database itself. Otherwise, the function
// will only retrieve the contents that are within the specified range, will create a new datatase containing the specified
// contents, and will return it.
func GetDatabase(databaseList skiplist.DBIndex[string, Database], databaseName string, ctx context.Context, start string, end string) (Database, error) {
	// Find the database first
	database, found := databaseList.Find(databaseName)
	if !found {
		return database, fmt.Errorf("failed to find database")
	}
	// Check to see if a range is provided
	if start == "" && end == "" {
		return database, nil
	}
	// only executes if we're trying to query a range of documents
	var results []contents.Document
	var err error

	// Querying over the specified range
	results, err = database.Documents.Query(ctx, start, end)
	if err != nil {
		return database, fmt.Errorf("failed to query documents")
	}

	// Set the queried documents back to the database (without modifying the skiplist)
	newDocuments := skiplist.NewSkipList[string, contents.Document]()
	for _, doc := range results {
		newDocuments.Upsert(doc.Name, func(key string, currValue contents.Document, exists bool) (contents.Document, error) {
			return doc, nil
		})
	}

	// Returns a database only containing the contents in the given range
	return Database{
		Name:      database.Name,
		Documents: newDocuments,
	}, nil
}

// PutDatabase takes a list of databases and a database name as input, and creates this new
// database and adds it to its respective databaseList. The function updateCheck is used to
// check if a database with the given databaseName already exists in the provided databaseList,
// and returns an error if this occurs.
//
// The function creates the name and the empty skiplist used for the database's document list,
// and then calls Upsert to add the newly created database to its respective skiplist.
func PutDatabase(databaseList skiplist.DBIndex[string, Database], databaseName string, subscriberHandler *sse.SubscriberHandler, fullPath string) (bool, error) {
	// Check if a database with databaseName already exists in databaseList
	updateCheck := func(key string, currValue Database, exists bool) (newValue Database, err error) {
		if exists {
			return currValue, fmt.Errorf("database already exists")
		}
		// Assign the name and initialize the documentList for the database
		newValue.Name = key
		newValue.Documents = skiplist.NewSkipList[string, contents.Document]()
		return newValue, nil
	}
	// Upsert the database into the given databaseList
	subscriberHandler.Notify(fullPath, "update", `{"path":"`+fullPath+`"}`)
	return databaseList.Upsert(databaseName, updateCheck)
}

// DeleteDatabase removes the given database from the provided databaseList skiplist.
// If the removal is successful, the function returns true and a nil value for error.
// If the removal is unsuccessful, the function returns false and an error messae.
func DeleteDatabase(databaseList skiplist.DBIndex[string, Database], database Database, subscriberHandler *sse.SubscriberHandler, fullPath string) (bool, error) {
	_, removed := databaseList.Remove(database.Name)
	// Check to see that the removal was successful
	if !removed {
		return false, fmt.Errorf("database %s could not be removed", database.Name)
	}
	subscriberHandler.Notify(fullPath, "delete", strconv.Quote(fullPath))

	return true, nil
}
