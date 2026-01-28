package skiplist

import (
	"context"
	"strconv" // Make sure you import strconv for int to string conversion
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSkipList(t *testing.T) {
	skiplist := NewSkipList[string, int]()
	assert.NotNil(t, skiplist, "SkipList should be initialized")
	assert.NotNil(t, skiplist.head, "SkipList should have a head node")
	assert.NotNil(t, skiplist.tail, "SkipList should have a tail node")
}

// Example of a fixed line
func TestInsertSkipList(t *testing.T) {
	skiplist := NewSkipList[string, int]()
	for i := 0; i < 10; i++ {
		key := strconv.Itoa(i) // Proper int to string conversion
		_, err := skiplist.Upsert(key, func(key string, currValue int, exists bool) (newValue int, err error) {
			return i, nil
		})
		assert.NoError(t, err, "Upsert should not return an error")
	}
}

func TestSkipListUpsertAndFind(t *testing.T) {
	skiplist := NewSkipList[string, int]()

	// Define an update check function
	updateCheck := func(key string, currValue int, exists bool) (newValue int, err error) {
		if exists {
			return currValue + 1, nil
		}
		return 1, nil
	}

	// Insert a value into the skiplist
	updated, err := skiplist.Upsert("key1", updateCheck)
	assert.NoError(t, err, "Upsert should not return an error")
	assert.True(t, updated, "Upsert should insert a new node")

	// Find the value
	val, found := skiplist.Find("key1")
	assert.True(t, found, "Key1 should be found")
	assert.Equal(t, 1, val, "Key1 should have value 1")

	// Upsert to update the value
	updated, err = skiplist.Upsert("key1", updateCheck)
	assert.NoError(t, err, "Upsert should not return an error when updating")
	assert.True(t, updated, "Upsert should update an existing node")

	// Verify the updated value
	val, found = skiplist.Find("key1")
	assert.True(t, found, "Key1 should be found after update")
	assert.Equal(t, 2, val, "Key1 should have updated value 2")
}
func TestSkipListRemove(t *testing.T) {
	skiplist := NewSkipList[string, int]()

	// Define an update check function
	updateCheck := func(key string, currValue int, exists bool) (newValue int, err error) {
		if exists {
			return currValue + 1, nil
		}
		return 1, nil
	}

	// Insert values into the skiplist
	_, _ = skiplist.Upsert("key1", updateCheck)
	_, _ = skiplist.Upsert("key2", updateCheck)

	// Remove key1
	removedValue, removed := skiplist.Remove("key1")
	assert.True(t, removed, "Key1 should be removed")
	assert.Equal(t, 1, removedValue, "Removed value should be 1")

	// Verify key1 no longer exists
	_, found := skiplist.Find("key1")
	assert.False(t, found, "Key1 should not be found after removal")

	// Verify key2 still exists
	val, found := skiplist.Find("key2")
	assert.True(t, found, "Key2 should still be found")
	assert.Equal(t, 1, val, "Key2 should have value 1")
}

func TestSkipListQuery(t *testing.T) {
	skiplist := NewSkipList[string, int]()

	// Define an update check function
	updateCheck := func(key string, currValue int, exists bool) (newValue int, err error) {
		if exists {
			return currValue + 1, nil
		}
		return 1, nil
	}

	// Insert values into the skiplist
	_, _ = skiplist.Upsert("key1", updateCheck)
	_, _ = skiplist.Upsert("key2", updateCheck)
	_, _ = skiplist.Upsert("key3", updateCheck)

	// Perform a query between key1 and key3
	ctx := context.TODO()
	results, err := skiplist.Query(ctx, "key1", "key3")
	assert.NoError(t, err, "Query should not return an error")
	assert.Equal(t, 3, len(results), "Query should return 3 results")

	// Check the values in the query result
	assert.Equal(t, 1, results[0], "Key1 should have value 1")
	assert.Equal(t, 1, results[1], "Key2 should have value 1")
	assert.Equal(t, 1, results[2], "Key3 should have value 1")
}
func TestSkipListQueryWithEndRange(t *testing.T) {
	skiplist := NewSkipList[string, int]()

	// Define an update check function
	updateCheck := func(key string, currValue int, exists bool) (newValue int, err error) {
		return currValue + 1, nil
	}

	// Insert values
	_, _ = skiplist.Upsert("a", updateCheck)
	_, _ = skiplist.Upsert("b", updateCheck)
	_, _ = skiplist.Upsert("c", updateCheck)

	// Query with range
	ctx := context.TODO()
	results, err := skiplist.Query(ctx, "a", "b")
	assert.NoError(t, err, "Query should not return an error")
	assert.Equal(t, 2, len(results), "Query should return 2 results")

	// Check the values
	assert.Equal(t, 1, results[0], "First value should be 1")
	assert.Equal(t, 1, results[1], "Second value should be 1")
}

func TestSkipListConcurrency(t *testing.T) {
	skiplist := NewSkipList[string, int]()
	ctx := context.TODO()

	// Define a check function for upsert
	updateCheck := func(key string, currValue int, exists bool) (newValue int, err error) {
		if exists {
			return currValue + 1, nil
		}
		return 1, nil
	}

	// Concurrent Upserts
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = skiplist.Upsert("key"+strconv.Itoa(i), updateCheck) // Fixed: strconv.Itoa(i) for string conversion
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify the skiplist contains all inserted keys
	for i := 0; i < 100; i++ {
		_, found := skiplist.Find("key" + strconv.Itoa(i)) // Fixed: strconv.Itoa(i)
		assert.True(t, found, "Key should be found after concurrent upserts")
	}

	// Concurrent Queries
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results, err := skiplist.Query(ctx, "key0", "key"+strconv.Itoa(i)) // Fixed: strconv.Itoa(i)
			assert.NoError(t, err, "Query should not return an error")
			assert.True(t, len(results) > 0, "Query should return results")
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
}
