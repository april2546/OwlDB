// Package auth implements the functions needed to authenticate and authorize users and
// create tokens for each user. This package also has functions for logging in/out, and
// a handler function for these requests.

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// A token is a struct that hold the username of a user, their corresponding
// token needed for login, and the time that their token will expire.
// A token will expire after 1 hour of it being generated.
type Token struct {
	Username   string    // name of the user associated with the token
	Token      string    // token associated with the username
	Expiration time.Time // indicates when the token will expire
}

// AuthManager manages users and tokens. This struct contains the duration that
// tokens should last, a map of strings that represent the tokens, and a map of
// strings that represents the user tokens needed for regenerating tokens
type AuthManager struct {
	tokenDuration time.Duration     // indicates how long the token will be valid for
	tokens        map[string]Token  // tokens used for autohrization
	userTokens    map[string]string // usernames that are associated with valid tokens
	mu            sync.Mutex        // controls access to tokens and usernames
}

// NewAuthManager creates a new AuthManager with a specified token expiration duration.
// It uses the provided expiration duration for the tokenDuration field, and initializes
// empty maps (string->token for tokens and string->string for user tokens) for the other
// two fields.
func NewAuthManager(tokenDuration time.Duration) *AuthManager {
	return &AuthManager{
		tokenDuration: tokenDuration,
		tokens:        make(map[string]Token),
		userTokens:    make(map[string]string),
	}
}

// LoadUsers will take a JSON File as input, and will then map the provided
// users to a token that the user will be able to use for authentication.
func (am *AuthManager) LoadUsers(filePath string) error {
	// Reading file inputs
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}
	// Attempting to unmarshall and store the users in the empty userTokens map
	var userTokens map[string]string
	if err := json.Unmarshal(file, &userTokens); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	// Load each user token into the AuthManager
	am.mu.Lock()
	defer am.mu.Unlock()
	for username, token := range userTokens {

		am.userTokens[username] = token
		am.tokens[token] = Token{
			Username:   username,
			Token:      token,
			Expiration: time.Now().Add(24 * time.Hour), // Never expires for predefined tokens
		}
	}
	return nil
}

// Login takes in a string representing a username and generates a new token
// for this user to use when authenticating. If a token already exists for
// the given user, then the function will delete the old token and will
// replace it with a new randomly generated token.
func (am *AuthManager) Login(username string) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if the user has an existing token and remove it
	if oldToken, exists := am.userTokens[username]; exists {
		delete(am.tokens, oldToken) // Remove the old token from the tokens map
	}

	// Generate a new token
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create a token entry with expiration time (1 hour)
	expiration := time.Now().Add(am.tokenDuration)
	newToken := Token{
		Username:   username,
		Token:      token,
		Expiration: expiration,
	}

	// Store the token
	am.tokens[token] = newToken
	am.userTokens[username] = token

	return token, nil
}

// Logout will take a token as input, and will remove the token from
// the token list, which will forfeit its access and will log the
// corresponding user out.
//
// If the given token does not exist, then the function throws and
// returns an error.
func (am *AuthManager) Logout(token string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.tokens[token]; !exists {
		return errors.New("invalid token")
	}

	delete(am.tokens, token)
	return nil
}

// Authenticate will take a token as input, and will check if the given
// token is valid and that the token has not expired. If the token exists,
// the function will extend the expiration time for the token according to
// the tokenDuration field in the provided AuthManager.
//
// If the token doesn't exist, an error will be thrown.
func (am *AuthManager) Authenticate(token string) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	t, exists := am.tokens[token]
	if !exists || time.Now().After(t.Expiration) {
		return "", errors.New("missing or invalid bearer token")
	}

	// Refresh token expiration
	t.Expiration = time.Now().Add(am.tokenDuration)
	am.tokens[token] = t

	return t.Username, nil
}

// generateToken generates a new random token that can be used for login.
func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Middleware is an HTTP middleware for token-based authentication and authorization.
// This function sets the CORS headers, and implements the appropriate HTTP Handlers.
//
// If the request is an OPTIONS request, then the function will bypass the login process.
// Otherwise, the function will get the Bearer token from the Authorization header,
// validate the format of the token, authenticate the Bearer token, and will add the
// corresponding user information to the request context.
//
// If there is no Authorization header, the Bearer token does not match the proper
// bearer token format, or the Bearer token is unable to be authenticated, then
// the function returns an Unauthorized status code.
func (am *AuthManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the CORS headers for all requests, including OPTIONS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Allow OPTIONS requests to bypass token validation (CORS preflight)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Allow login requests without a token
		if r.URL.Path == "/auth" && r.Method == http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		// Extract the Bearer token from the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing or invalid bearer token", http.StatusUnauthorized)
			return
		}

		// Check if the token follows the "Bearer <token>" format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			http.Error(w, "Missing or invalid bearer token", http.StatusUnauthorized)
			return
		}
		token := tokenParts[1]

		// Authenticate the token
		username, err := am.Authenticate(token)
		if err != nil {
			http.Error(w, "Missing or invalid bearer token", http.StatusUnauthorized)
			return
		}

		// Attach the username to the request context and proceed
		ctx := r.Context()
		ctx = contextWithUsername(ctx, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// contextWithUsername adds the username to the request context.
// This is used in the Middleware function. It takes a string
// representing a username and the associated request context as input.
func contextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, "username", username)
}

// UsernameFromContext extracts the username from the request context.
// It takes a request context as input and returns a string representing
// the request's associated username.
func UsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// AuthHandler manages HTTP requests for authentication.
type AuthHandler struct {
	authManager *AuthManager
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authManager *AuthManager) *AuthHandler {
	return &AuthHandler{authManager: authManager}
}

// HandleRequest sets the CORS headers, and handles all login, logout, and OPTIONS requests.
// If an unexpected request occurs, the function returns a 405 Status code.
func (ah *AuthHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for Swagger UI and other cross-origin clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Allow", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Methods", "POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	// Handling request types
	switch r.Method {
	case http.MethodPost:
		ah.LoginHandler(w, r)
	case http.MethodDelete:
		ah.LogoutHandler(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK) // Respond to OPTIONS request with status OK
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// LoginHandler is a function that handles the login requests. The function reads the request
// body, stores the information provided by the request into a struct, extracts the username,
// and then logs the user in using a randomly generated token.
//
// If the request body is empty, there is no provided username, or if the response data cannot
// be encoded, the function will then return a 401 Status code.
func (ah *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Decode the request body
	var requestData struct {
		Username string `json:"username"`
	}
	// Unmarshall the request body data
	if err := json.Unmarshal(bodyBytes, &requestData); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// Check if username is empty
	if requestData.Username == "" {
		http.Error(w, "username cannot be empty", http.StatusBadRequest)
		return
	}

	// Log in the user and generate a token
	token, err := ah.authManager.Login(requestData.Username)
	if err != nil {
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}

	// Respond with the token
	responseData := map[string]string{
		"token": token,
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(responseData)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// LogoutHandler handles all logout requests. The function takes a token from
// the Authorization header, and attempts to log out a user with the given token.
//
// If the authorization header is empty or the Bearer token is formatted improperly,
// then the function returns a 401 Status code
func (ah *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the token from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing or invalid bearer token", http.StatusUnauthorized)
		return
	}

	// Split the Authorization header to get the token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		http.Error(w, "Missing or invalid bearer token", http.StatusUnauthorized)
		return
	}
	token := tokenParts[1]

	// Log out the user (invalidate the token)
	err := ah.authManager.Logout(token)
	if err != nil {
		http.Error(w, "logout failed", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
