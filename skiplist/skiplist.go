// Package skiplist contains the struct that represents a skiplist and its nodes, and contains the
// methods needed to insert a node, remove a node, query contents, and upsert a node.

package skiplist

import (
	"cmp"
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const MAX_LEVEL = 11

type UpdateCheck[K cmp.Ordered, V any] func(key K, currValue V, exists bool) (newValue V, err error)

// This is the interface that holds all of the skiplist methods.
type DBIndex[K cmp.Ordered, V any] interface {
	Find(key K) (foundValue V, found bool)
	Upsert(key K, check UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedValue V, removed bool)
	Query(ctx context.Context, start K, end K) (results []V, err error)
}

// This struct is used to represent a list of databases, a list of documents,
// or a list of collections. It has a field representing the head of the list, a field for the tail,
// and a field that tracks the number of operations that have been performed on the skiplist.
// The count field is used in the Query method.
type SkipList[K cmp.Ordered, V any] struct {
	head  *Node[K, V] // beginning of the linked list
	tail  *Node[K, V] // tail of the linked list
	count atomic.Int64
}

// This struct is used as a node in a skiplist struct. It contains a mutex lock to control acess,
// a key (which is typically the name of the document/database/collection), and a value (typically the contents
// of the daabase/document/collection). The field topLevel indicates how many levels a skiplist will have,
// the field marked is used to check if a node will be deleted, ________, and next points to the node's
// next neighbor in the skiplist.
type Node[K cmp.Ordered, V any] struct {
	mutex       sync.Mutex
	key         K
	value       atomic.Pointer[V]
	topLevel    int
	marked      atomic.Bool
	fullyLinked atomic.Bool
	next        []atomic.Pointer[Node[K, V]]
}

// NewSkipList initializes a new empty skiplist. It initializes the head and tail nodes
// for every level in the skiplist, sets the pointers for these nodes, and sets the
// atomic count to zero.
func NewSkipList[K cmp.Ordered, V any]() *SkipList[K, V] {
	rand.Seed(time.Now().UnixNano())

	// Create a new skip list with level set to 0 initially
	skipList := &SkipList[K, V]{}

	//create tail node
	tailNode := &Node[K, V]{
		next:     make([]atomic.Pointer[Node[K, V]], MAX_LEVEL),
		topLevel: MAX_LEVEL,
	}
	// Create a head node for this level
	headNode := &Node[K, V]{
		next:     make([]atomic.Pointer[Node[K, V]], MAX_LEVEL),
		topLevel: MAX_LEVEL,
	}

	// Initialize the head and tail nodes for all levels
	for i := 0; i < MAX_LEVEL; i++ {
		// Set the head's next pointer to point to the tail node at this level
		headNode.next[i].Store(tailNode)
	}

	// Initialize the atomic count to zero
	skipList.count.Store(0)
	skipList.head = headNode
	skipList.tail = tailNode
	return skipList
}

// randomLevel generates a random level for node insertion
func randomLevel() int {
	rand.Seed(time.Now().UnixNano())
	level := 1
	for rand.Float64() < 0.5 && level < MAX_LEVEL {
		level++
	}
	return level
}

// Upsert will either update a skiplist or it will insert a new node into the provided skiplist.
// For updating, it finds the node, checks that the node is still a part of the skiplist, and then updates the node's
// value. If the node has been marked for removal or is no longer fully linked, it retries the operation.
// For insertion, it generates a random top level for the new node, locks the necessary predecessors, and inserts the
// new node in the correct position for every level in the skiplist. The function calls a check
// function to get the new value for both updates and inserts. If the check function fails, then the fuction returns
// false and an error. The skiplist's count field is incremented upon successful updating/insertion, and the function
// returns true with a nil error value.
func (skiplist *SkipList[K, V]) Upsert(key K, check UpdateCheck[K, V]) (updated bool, err error) {
	// Start by finding the key in the skiplist
	for {
		levelFound, preds, succs := skiplist.find(key)
		// Map to track the nodes we've locked
		lockedNodes := make(map[*Node[K, V]]bool)

		// If the key is found, attempt to update the value
		if levelFound != -1 {
			foundNode := succs[levelFound]

			// Lock the node for update
			foundNode.mutex.Lock()
			lockedNodes[foundNode] = true

			// Check if the node is still valid (not marked for removal and fully linked)
			if foundNode.marked.Load() || !foundNode.fullyLinked.Load() {
				foundNode.mutex.Unlock()
				lockedNodes[foundNode] = true
				continue // Retry, as the node was marked for removal or not fully linked
			}

			// Call the check function to determine the new value based on the current value

			newValue, err := check(key, *foundNode.value.Load(), true)
			if err != nil {
				foundNode.mutex.Unlock()
				delete(lockedNodes, foundNode)
				return false, err // Return if there was an error during the check
			}

			// Update the value and unlock the node

			foundNode.value.Store(&newValue)
			foundNode.mutex.Unlock()
			delete(lockedNodes, foundNode)
			return true, nil // Successfully updated the node
		}

		// If the key is not found, proceed to insert a new node
		topLevel := randomLevel() // Choose a random top level for the new node

		// Lock predecessors to ensure that we can safely insert the new node
		highestLocked := -1
		valid := true
		level := 0

		for valid && level <= topLevel {
			predNode := preds[level]
			succNode := succs[level]
			if predNode == nil || succNode == nil {
				valid = false
				break
			}

			if !lockedNodes[predNode] {
				predNode.mutex.Lock()
				lockedNodes[predNode] = true
			}
			highestLocked = level

			// Validate that the current state of the list hasn't changed
			unmarked := !predNode.marked.Load() && !succNode.marked.Load()
			connected := predNode.next[level].Load() == succNode
			valid = unmarked && connected
			level++
		}

		if !valid {
			for level := highestLocked; level >= 0; level-- {
				predNode := preds[level]
				if lockedNodes[predNode] {
					predNode.mutex.Unlock()
					delete(lockedNodes, predNode)
				}
			}
			continue
		}

		// Create a new node with the zero value of type V
		newNode := &Node[K, V]{
			key:         key,
			next:        make([]atomic.Pointer[Node[K, V]], topLevel+1),
			topLevel:    topLevel,
			fullyLinked: atomic.Bool{}, // Not fully linked yet
		}

		newNode.value.Store(new(V))
		// Call the check function with the zero value since the node doesn't exist, and exists=false
		newValue, err := check(key, *new(V), false)
		if err != nil {
			// If the check function returns an error, unlock and return
			for level := highestLocked; level >= 0; level-- {
				predNode := preds[level]
				if lockedNodes[predNode] {
					predNode.mutex.Unlock()
					delete(lockedNodes, predNode)
				}
			}
			return false, err
		}

		// Set the value returned by the check function
		newNode.value.Store(&newValue)

		// Insert the new node into the skip list at each level
		for level = 0; level <= topLevel; level++ {
			predNode := preds[level]
			predNode.next[level].Store(newNode)
			newNode.next[level].Store(succs[level])
		}
		// Mark the node as fully linked and unlock everything
		newNode.fullyLinked.Store(true)
		for level = highestLocked; level >= 0; level-- {
			predNode := preds[level]
			if lockedNodes[predNode] {
				predNode.mutex.Unlock()
				delete(lockedNodes, predNode)
			}
		}

		// Increment the SkipList count
		skiplist.count.Add(1)
		return true, nil // Successfully inserted the node
	}
}

// Remove attempts to remove the node indicated by the given key from its corresponding skiplist.
// It first locates the node to be removed with the find() helper function, which returns the node’s predecessors
// and successors.
// If the node is found, the function checks whether it is valid for removal (fully linked and not marked). The node is then
// marked for removal and locked. If the node isn't found, then the function returns an empty value and false indicating that
// the removal failed.
// If the node is already marked or invalid, the function returns an empty value and false, indicating
// that the removal failed. After marking, it locks all necessary predecessors and checks that the node is still present.
// The function then updates the pointers of the predecessors to skip over the node. The function unlocks the node
// and all predecessors upon completion, increments the skiplist’s counter, and returns the removed value and 'true' indicating
// the removal's success.
func (skiplist *SkipList[K, V]) Remove(key K) (removedValue V, removed bool) {
	var victim *Node[K, V]
	isMarked := false
	topLevel := -1

	// Keep trying to remove until success/failure
	for {
		// Use find() to locate the key and get the predecessors and successors
		foundLevel, preds, succs := skiplist.find(key)

		// If the key is not found
		if foundLevel == -1 {
			return *new(V), false
		}

		// Set victim as the node to remove
		victim = succs[foundLevel]
		if victim == nil {
			return *new(V), false
		}
		// Map to track the nodes we've locked
		lockedNodes := make(map[*Node[K, V]]bool)

		// First time through, mark and lock victim
		if !isMarked {
			if !victim.fullyLinked.Load() || victim.marked.Load() || victim.topLevel != foundLevel {
				return *new(V), false
			}

			topLevel = victim.topLevel
			victim.mutex.Lock()
			lockedNodes[victim] = true

			victim.marked.Store(true)
			isMarked = true
		}

		// Lock all predecessors
		highestLocked := -1
		level := 0
		valid := true
		for valid && (level <= topLevel) {
			pred := preds[level]
			// Lock predecessor if not already locked
			if !lockedNodes[pred] {
				pred.mutex.Lock()
				lockedNodes[pred] = true
			}
			highestLocked = level

			successor := pred.next[level].Load() == victim
			valid = !pred.marked.Load() && successor
			level++
		}

		// Unlock if not valid and retry
		if !valid {
			for level := highestLocked; level >= 0; level-- {
				pred := preds[level]
				if lockedNodes[pred] {
					pred.mutex.Unlock()
					delete(lockedNodes, pred)
				}
			}
			continue
		}

		// Remove the node by updating pointers
		for level := topLevel; level >= 0; level-- {
			preds[level].next[level].Store(victim.next[level].Load())
		}

		// Unlock victim and all predecessors
		if lockedNodes[victim] {
			victim.mutex.Unlock()
			delete(lockedNodes, victim)
		}
		for level := highestLocked; level >= 0; level-- {
			pred := preds[level]
			if lockedNodes[pred] {
				pred.mutex.Unlock()
				delete(lockedNodes, pred)
			}
		}

		removedValue = *victim.value.Load()
		removed = true
		// Increase atomic counter
		skiplist.count.Add(1)
		return removedValue, true
	}
}

// find is a helper function that is used to locate a node in a skiplist.
// It is used in the Find, Query, and Remove methods.
// find starts from the top level of the skiplist and traverses through the levels to find the node or its closest
// predecessor and successor nodes.
// The function returns the level where the key was found, a slice of predecessor nodes, and a slice of successor nodes.
// The predecessor nodes represent the nodes just before the key at each level, and the successor nodes represent
// the nodes just after the key.
// If the key is not found, the returned level is -1, and the function still provides the predeccessor and
// successor nodes.
func (s *SkipList[K, V]) find(key K) (int, []*Node[K, V], []*Node[K, V]) {
	preds := make([]*Node[K, V], MAX_LEVEL)
	succs := make([]*Node[K, V], MAX_LEVEL)

	foundLevel := -1
	pred := s.head

	level := MAX_LEVEL - 1
	for level >= 0 {
		curr := pred.next[level].Load()
		if curr != s.tail {
			for key > curr.key {
				pred = curr
				curr = pred.next[level].Load()
				if curr == s.tail {
					break
				}
			}
		}

		if foundLevel == -1 && key == curr.key {
			foundLevel = level
		}
		preds[level] = pred
		succs[level] = curr
		level = level - 1
	}
	return foundLevel, preds, succs
}

// Find takes in a key of a node as input, and returns the value of the node associated
// with the given key. If the given node is not found, then the function returns an empty
// value and 'false' indicating failure. If the node is found, then the function returns
// the value of the node, and the status of its marked and fully linked fields.
func (s *SkipList[K, V]) Find(key K) (foundValue V, found bool) {
	// Get the predecessor and successor nodes
	levelFound, _, succs := s.find(key)

	if levelFound == -1 {
		return *new(V), false
	}
	foundValue = *succs[levelFound].value.Load()
	return foundValue, succs[levelFound].fullyLinked.Load() && !succs[levelFound].marked.Load()
}

// Query takes in a context, a start, and an end value as input, and attempts to return
// the contents of a given skiplist.
// Query can either return everything inside a given skiplist, or it can return only the contents
// of a skiplist that lie withing a certain range. It will also return a boolean, true indicating success
// and false indicating failure.
// The inputs 'start' and 'end' indicate the range that the function will query over. If 'start' is an empty
// string, then the Query function will begin at the beginning of the skiplist. If 'end' is an empty string,
// then the function will query until the end of the skiplist. If both inputs are empty string,
// The function will query and return everything in the skiplist.
func (skipList *SkipList[K, V]) Query(ctx context.Context, start K, end K) (results []V, err error) {
	// Get the current count of the skip list before starting the query
	preCount := skipList.count.Load()

	// Find the node to start the traversal from
	var current *Node[K, V]
	// Type assertion to check if K is a string
	if startStr, ok := any(start).(string); ok && startStr == "" {
		// If start is an empty string, start from the head of the skip list
		current = skipList.head.next[0].Load()
	} else {
		// Otherwise, find the node to start from
		_, _, succs := skipList.find(start)
		//     for i, succNode := range succs {
		// 			if succNode != nil {
		// 				fmt.Printf("Level %d: succNode.key = %v\n", i, succNode.key)
		// 			} else {
		// 				fmt.Printf("Level %d: succNode is nil\n", i)
		// 			}
		// 		}
		//current = pred[len(pred)-1].next[0].Load()
		current = succs[0]
	}
	// Find the node to end the traversal from (successor of the start key)
	var goToEnd bool
	// Type assertion to check if K is a string
	if endStr, ok := any(end).(string); ok && endStr == "" {
		// If start is an empty string, start from the head of the skip list
		goToEnd = true
	} else {
		goToEnd = false
	}

	// Initialize results array
	results = []V{}

	// Traverse from the found node at level 0
	for current != skipList.tail {
		// Check if the context is done (to handle llations)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue the query as the context is valid
		}
		if !goToEnd {
			// If current key is greater than the end, stop the traversal
			if current.key > end {
				break
			}
		}

		// Collect values if the node is fully linked and not marked for removal
		if current.fullyLinked.Load() && !current.marked.Load() {
			results = append(results, *current.value.Load())
		}

		// Move to the next node in the skip list at level 0
		current = current.next[0].Load()
	}

	// Get the current count of the skip list after the query
	postCount := skipList.count.Load()

	// If the count has changed during the query, retry the query
	if postCount != preCount {
		return skipList.Query(ctx, start, end) // Retry the query
	}

	return results, nil
}
