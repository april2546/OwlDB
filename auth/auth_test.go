package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAuthManager(t *testing.T) {
	am := NewAuthManager(time.Hour)
	assert.NotNil(t, am)
	assert.Equal(t, time.Hour, am.tokenDuration)
	assert.Empty(t, am.tokens)
	assert.Empty(t, am.userTokens)
}

func TestLogin(t *testing.T) {
	am := NewAuthManager(time.Hour)

	// Test logging in a user
	token, err := am.Login("user1")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Ensure the token is stored correctly
	am.mu.Lock()
	defer am.mu.Unlock()

	storedToken, exists := am.tokens[token]
	assert.True(t, exists)
	assert.Equal(t, "user1", storedToken.Username)
}

func TestLogout(t *testing.T) {
	am := NewAuthManager(time.Hour)

	// Log in a user
	token, err := am.Login("user1")
	assert.NoError(t, err)

	// Log out the user
	err = am.Logout(token)
	assert.NoError(t, err)

	// Ensure the token is removed
	am.mu.Lock()
	defer am.mu.Unlock()

	_, exists := am.tokens[token]
	assert.False(t, exists)
}

func TestAuthenticate(t *testing.T) {
	am := NewAuthManager(time.Hour)

	// Log in a user
	token, err := am.Login("user1")
	assert.NoError(t, err)

	// Authenticate the token
	username, err := am.Authenticate(token)
	assert.NoError(t, err)
	assert.Equal(t, "user1", username)

	// Test with an invalid token
	_, err = am.Authenticate("invalid-token")
	assert.Error(t, err)
}

func TestMiddleware(t *testing.T) {
	am := NewAuthManager(time.Hour)

	// Log in a user
	token, err := am.Login("user1")
	assert.NoError(t, err)

	// Define a next handler to check the context
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, ok := UsernameFromContext(r.Context())
		assert.True(t, ok)
		assert.Equal(t, "user1", username)
		w.WriteHeader(http.StatusOK)
	})

	// Create a request and attach the Authorization header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Use middleware
	rr := httptest.NewRecorder()
	middleware := am.Middleware(nextHandler)
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleLoginRequest(t *testing.T) {
	am := NewAuthManager(time.Hour)
	ah := NewAuthHandler(am)

	// Create a login request
	requestBody, _ := json.Marshal(map[string]string{
		"username": "user1",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// Serve the request using the auth handler
	ah.HandleRequest(rr, req)

	// Verify the response
	assert.Equal(t, http.StatusOK, rr.Code)

	var responseData map[string]string
	err := json.NewDecoder(rr.Body).Decode(&responseData)
	assert.NoError(t, err)

	// Verify token in the response
	token, exists := responseData["token"]
	assert.True(t, exists)
	assert.NotEmpty(t, token)

	// Ensure the token is valid
	username, err := am.Authenticate(token)
	assert.NoError(t, err)
	assert.Equal(t, "user1", username)
}

func TestHandleLogoutRequest(t *testing.T) {
	am := NewAuthManager(time.Hour)
	ah := NewAuthHandler(am)

	// Log in a user
	token, err := am.Login("user1")
	assert.NoError(t, err)

	// Create a logout request
	req := httptest.NewRequest(http.MethodDelete, "/auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	// Serve the request using the auth handler
	ah.HandleRequest(rr, req)

	// Verify the response
	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Ensure the token is invalidated
	_, err = am.Authenticate(token)
	assert.Error(t, err)
}

func TestHandleOptionsRequest(t *testing.T) {
	ah := NewAuthHandler(NewAuthManager(time.Hour))

	// Create an OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/auth", nil)
	rr := httptest.NewRecorder()

	// Serve the request using the auth handler
	ah.HandleRequest(rr, req)

	// Verify the response
	assert.Equal(t, http.StatusOK, rr.Code)
}
