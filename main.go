package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/RICE-COMP318-FALL24/owldb-p1group32/auth"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/handlers"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/jsondata"
	"github.com/RICE-COMP318-FALL24/owldb-p1group32/skiplist"
	sse "github.com/RICE-COMP318-FALL24/owldb-p1group32/sse"
)

func main() {
	// command-line flags (-p, -s, -t)
	portnum := flag.String("p", "3318", "Port to listen on")
	jsonFlag := flag.String("s", "", "Name of file with JSON schema")
	tokenFlag := flag.String("t", "", "JSON file with mapping of usernames to tokens")
	flag.Parse()

	// ensure a file with json schema is named
	if *jsonFlag == "" {
		log.Fatal("Error: Must specify the name of a file with the JSON schema using the -s flag\n")
	}
	// ensure mapping of usernames to tokens is given
	if *tokenFlag == "" {
		log.Fatal("Error: Must specify the JSON file with mapping of user names to tokens using the -t flag\n")
	}

	schem, err := jsondata.New(*jsonFlag)
	if err != nil {
		log.Fatal("Error: Provided schema could not be compiled\n")
	}

	port, err := strconv.Atoi(*portnum)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Starting server on port %d...\n", port)

	//Create the AuthManager with 1-hour token expiration
	authManager := auth.NewAuthManager(1 * time.Hour)

	// Load the user tokens from a file
	if err := authManager.LoadUsers(*tokenFlag); err != nil {
		log.Fatal(err)
	}

	authHandler := auth.NewAuthHandler(authManager)

	// Initialize the SubscriberHandler and SupscriptionFactory
	subscriptionFactory := func() sse.DBIndex[string, *sse.Subscriber] {
		// Assuming a SkipList implementation exists
		return skiplist.NewSkipList[string, *sse.Subscriber]()
	}
	subscriberHandler := sse.NewSubscriberHandler(
		// Resource to token mapping
		skiplist.NewSkipList[string, sse.DBIndex[string, *sse.Subscriber]](),
		subscriptionFactory,
	)
	// Create the auth handlers
	mux := http.NewServeMux()
	mux.Handle("/auth", http.HandlerFunc(authHandler.HandleRequest))

	// Set up the /subscribe route using the SSE handler
	mux.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		resource := r.URL.Query().Get("resource")
		token := r.URL.Query().Get("token")
		subscriberHandler.SSEHandler(w, r, resource, token)
	})
	databaseList := handlers.New(&schem, subscriberHandler)

	// Protected routes (requires token-based authentication)
	// Wrap the /v1/ endpoint with the auth middleware for database access
	mux.Handle("/v1/", authManager.Middleware(http.HandlerFunc(databaseList.V1Handler)))

	// initialize server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// The following code should go last and remain unchanged.
	// Note that you must actually initialize 'server' and 'port'
	// before this.  Note that the server is started below by
	// calling ListenAndServe.  You must not start the server
	// before this.

	// signal.Notify requires the channel to be buffered
	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, os.Interrupt, syscall.SIGTERM)
	go func() {
		// Wait for Ctrl-C signal
		<-ctrlc
		server.Close()
	}()

	// Start server
	slog.Info("Listening", "port", port)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed", "error", err)
	} else {
		slog.Info("Server closed", "error", err)
	}
}
