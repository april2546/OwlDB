// Package sse implements the structs used to represent a subscription
// manager and a subscriber handler which stores the subscription information
// to be used in other files. It also implements the functions NewSubscriberHandler,
// commentSender, updateEventSender, deleteEventSender, subscribePath,
// Notify, addSubscription, deleteSusbcription, and the SSEHandler. The SSEHandler
// implementes the lowercased methods to aid in managing subscriptions for databases,
// documents, and collections.

package sse

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
)

// This is the DBIndex interface which we create in the skiplist package and used here
// to track the subscriptions.
type DBIndex[K cmp.Ordered, V any] interface {
	Find(key K) (foundValue V, found bool)
	Upsert(key K, check skiplist.UpdateCheck[K, V]) (updated bool, err error)
	Remove(key K) (removedValue V, removed bool)
	Query(ctx context.Context, start K, end K) (results []V, err error)
}

type writeFlusher interface {
	http.ResponseWriter
	http.Flusher
}

// SubscriberManager struct manages subscriptions by storing the path to the db/doc/col,
// the event as a channel, as well as the context to keep all of the information regarding
// writers and readers accessible.
type Subscriber struct {
	path  string
	event chan string
	ctx   context.Context
}

// The SubscriberFactory creates a skiplist of DBIndex interface mapping the path to the
// subscriber types with the above 3 fields.
type SubscriberFactory func() DBIndex[string, *Subscriber]

// The SubscriberHandler struct manages and facilitates subscriptions by mapping resource
// tokens to collections of subscribers through a DBIndex, utilizing a SubscriberFactory to
// generate new subscriptions as needed.
type SubscriberHandler struct {
	resourceToken       DBIndex[string, DBIndex[string, *Subscriber]]
	subscriptionFactory SubscriberFactory
}

// NewSubscriberManager initializes the subscriber manager with a subscription handler
// to be used where it needs to be outside of this package.
func NewSubscriberHandler(resourceTotoken DBIndex[string, DBIndex[string, *Subscriber]], subscriptionFactory SubscriberFactory) *SubscriberHandler {
	return &SubscriberHandler{resourceToken: resourceTotoken, subscriptionFactory: subscriptionFactory}
}

func commentSender(wf writeFlusher) {
	var evt bytes.Buffer
	evt.WriteString(fmt.Sprintf(": this is a comment message\n"))

	slog.Info("Sending", "comment", evt.String())

	// Send event
	wf.Write(evt.Bytes())
	wf.Flush()
}

// updateEventSender sends an "update" event message to the specified writeFlusher, typically used in an SSE
// setup to notify subscribers of data updates. The function logs the event details, constructs the event with an "update"
// label, assigns a unique ID based on the current time in milliseconds, and includes the provided data string. The event
// is then written to the writeFlusher and flushed to ensure it is immediately transmitted to clients.
func updateEventSender(wf writeFlusher, data string) {
	var evt bytes.Buffer

	slog.Info("Sending", "event", "update", "data", data)

	// Writing subscription information
	evt.WriteString(fmt.Sprintf("event: %s\n", "update"))
	evt.WriteString(fmt.Sprintf("id: %d\n", time.Now().UnixMilli()))
	evt.WriteString(fmt.Sprintf("data: %v\n\n", data))

	// Send event
	wf.Write(evt.Bytes())
	wf.Flush()
}

// deleteEventSender sends a "delete" event message to the specified writeFlusher, typically for notifying subscribers
// of a deletion in an SSE context. The function logs the event details, constructs the event with
// a "delete" label, assigns a unique ID based on the current time in milliseconds, and includes the provided data string.
// The event is then written to the writeFlusher and flushed to ensure immediate transmission to clients.
func deleteEventSender(wf writeFlusher, data string) {
	var evt bytes.Buffer

	// Add quotations if the event is a delete event
	slog.Info("Sending", "event", "delete", "data", data)

	// Writing subscription information
	evt.WriteString(fmt.Sprintf("event: %s\n", "delete"))
	evt.WriteString(fmt.Sprintf("id: %d\n", time.Now().UnixMilli()))
	evt.WriteString(fmt.Sprintf("data: %v\n\n", data))

	// Send event
	wf.Write(evt.Bytes())
	wf.Flush()
}

// SubscribePath registers a subscription for a given resource path in the SubscriberHandler's resourceToken map.
// The function initializes a new subscriber using the subscriptionFactory above and attempts to insert the resource
// path into the skiplist if it does not already exist.
//
// The UPSERT function checks if the resource path exists;
// if it does, it returns the current value without creating a duplicate. If the path is new, it inserts
// the new subscriber. If there is an error during insertion or if the path is updated, the function
// returns the error; otherwise, it completes successfully.
func (sh *SubscriberHandler) SubscribePath(resource string) error {
	newSubscriber := sh.subscriptionFactory() // Initialize new subscriber
	// Insert the new resource path to skiplist if one is not found
	updated, err := sh.resourceToken.Upsert(resource, func(key string, currValue DBIndex[string, *Subscriber], exists bool) (DBIndex[string, *Subscriber], error) {
		if exists {
			//return currValue, errors.New("resource has been subscribed already")
			return currValue, nil // If the path already exists, just return the current value (skiplist)
		}
		return newSubscriber, nil
	})

	if err != nil || updated {
		return err
	}

	return nil
}

// Notify sends a specified event and data to all subscribers of a given resource path in the SubscriberHandler.
// The function splits the resource path and constructs paths to notify incrementally. For each path segment,
// it checks if there are subscribers by querying the resourceToken skiplist.
//
// If a subscription is found,
// it retrieves all active subscriptions and sends the formatted event and data to each subscriber's event channel.
// If a channel is full, it logs the path but continues processing other subscriptions. This allows hierarchical
// notifications for resources, handling both document and collection-level subscriptions.
func (sh *SubscriberHandler) Notify(resource string, event string, data string) {
	slog.Info("starting notifying", "resource", resource)
	rawparts := strings.Split(resource, "/")
	pathToBuild := ""
	for i, path := range rawparts {
		pathToBuild = pathToBuild + "/" + path
		pathToBuild = strings.TrimPrefix(pathToBuild, "/")
		fmt.Println("index i : ", i, " pathTOBuild: ", pathToBuild)
		if i%2 == 0 || i == (len(rawparts)-1) {
			pathSubscription, found := sh.resourceToken.Find(pathToBuild)
			fmt.Println("THERE IS SUBSCRIPTION ON THIS PATH")
			fmt.Println("This is the skiplist of tokens associated with this path: ", pathSubscription)

			if !found {
				slog.Info(path)
				slog.Info("there is no subscription on this path")
				continue
			}

			fmt.Println("calling query for context.background")
			subscriptons, err := pathSubscription.Query(context.Background(), "", "")
			if err != nil {
				slog.Info("fail to query path subscriptions", "path", path)
				continue
			}
			fmt.Println("These are the subscriptions: ", subscriptons)

			for _, subscription := range subscriptons {
				select {
				case subscription.event <- fmt.Sprintf("%s;%s", event, data):
					fmt.Println("Sent event: ", event, " and data: ", data)

					slog.Info("successfully sent event and data")
				default:
					slog.Info("channel is full", "path", path)
				}
			}
		}
	}
}

// addSubscription adds a new subscriber for a specific resource path using the provided token in the SubscriberHandler.
// The function first checks if the resource path exists in the resourceToken skiplist.
//
// If not, it returns an error,
// indicating that no prior subscriptions exist for the path. If the path is found, it creates a new Subscriber with
// a buffered event channel and the HTTP request's context. The function then attempts to insert this new subscription
// into the skiplist using the token as the key. If a subscription with the same token already exists, it returns an
// error; otherwise, it adds the subscription successfully.
func (sh *SubscriberHandler) addSubscription(resource string, token string, r *http.Request) error {
	pathSubscription, found := sh.resourceToken.Find(resource)

	if !found {
		slog.Info("the given resource path is not yet subscribed")
		return errors.New("the given resource path is not yet subscribed")
	}

	subscription := &Subscriber{path: resource, event: make(chan string, 100), ctx: r.Context()}
	slog.Info("start adding new subscription")
	updated, err := pathSubscription.Upsert(token, func(key string, currValue *Subscriber, exists bool) (*Subscriber, error) {
		if exists {
			return currValue, errors.New("the subscription already exists")
		}
		return subscription, nil
	})

	if err != nil || updated {
		return err
	}

	return nil
}

// deleteSubscription removes a subscriber for a specific resource path in the SubscriberHandler using the provided token.
// The function first checks if the resource path exists in the resourceToken skiplist. If the path is not subscribed,
// it returns an error indicating that no subscriptions are present for the path. If the path exists, it attempts
// to remove the subscription associated with the given token. If removal is successful, the function completes without
// error; if the subscription could not be removed, it returns an error.

func (sh *SubscriberHandler) deleteSubscription(resource string, token string) error {
	pathSubscription, found := sh.resourceToken.Find(resource)
	if !found {
		slog.Info("the given resource path is not yet subscribed")
		return errors.New("the given resource path is not yet subscribed")
	}

	_, removed := pathSubscription.Remove(token)
	if !removed {
		return errors.New("fail to remove the subscription")
	}

	return nil
}

// SSEHandler manages an SSE connection for a client subscribing to a specific resource path in
// the SubscriberHandler. It begins by looking at the resource path and if it is subscribed and adds a new subscription using the
// given token. If successful, it sets up the HTTP headers for an SSE connection, confirming the client is
// connected with an initial update event. A ticker is then initialized to send keep-alive comments every 15 seconds.
// The function listens for various events: on "put"/"update" or "delete" events received via the subscription’s event channel,
// it forwards these to the client using outlined event-sending functions. When the client's context signals a disconnection,
// SSEHandler removes the subscription and stops, allowing for disconnection and resource cleanup.
func (sh *SubscriberHandler) SSEHandler(w http.ResponseWriter, r *http.Request, resource string, token string) {
	err := sh.SubscribePath(resource)
	if err != nil {
		slog.Info(err.Error())
	}

	err = sh.addSubscription(resource, token, r)
	if err != nil {
		slog.Info(err.Error())
	}
	slog.Info(resource)

	// Get the subscription
	subPath, exists := sh.resourceToken.Find(resource)

	if !exists {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "failed to get path subscriptions", http.StatusInternalServerError)
		return
	}
	subscription, exists := subPath.Find(token)

	if !exists {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "fail to get the subscription based on the token", http.StatusInternalServerError)
		return
	}

	wf, ok := w.(writeFlusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	slog.Info("Converted to writeFlusher")

	slog.Info("In header setup")

	// Set up SSE headers
	wf.Header().Set("Content-Type", "text/event-stream")
	wf.Header().Set("Cache-Control", "no-cache")
	wf.Header().Set("Connection", "keep-alive")
	wf.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
	wf.Header().Set("Access-Control-Allow-Origin", "*")
	wf.WriteHeader(http.StatusOK)
	wf.Flush()

	updateEventSender(wf, "\"Successfully connected!\"")

	// Keep the connection alive and collect subscriptions until closed
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			commentSender(wf)
			wf.Flush()
		case eventData := <-subscription.event:
			slog.Info("eventData", "is", eventData)
			parts := strings.SplitN(eventData, ";", 2)
			event := parts[0]
			data := parts[1]
			if event == "update" {
				updateEventSender(wf, data)
			} else {
				deleteEventSender(wf, data)
			}
		case <-subscription.ctx.Done():
			// Remove the subscription when the client disconnects
			sh.deleteSubscription(resource, token)
			slog.Info("Client closed connection")

		}
	}
	// Otherwise don't do anything with subscription
}
