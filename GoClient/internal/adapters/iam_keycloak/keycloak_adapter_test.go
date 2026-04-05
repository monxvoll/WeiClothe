package iam_keycloak

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestValidateToken_Success(t *testing.T) {
	// 1. Configure the "Mock Keycloak"
	// This server will intercept the adapter's request and return whatever we tell it to.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the adapter sent the token correctly in the Header
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"sub": "123e4567-e89b-12d3-a456-426614174000", "email": "test@weiclothe.com"}`)
	}))
	defer mockServer.Close()

	// 2. Instantiate the adapter, and pass it the mock server URL
	adapter := NewKeycloakAdapter(mockServer.URL, "weiclothe", "dummy-client", "dummy-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute the function
	uid, err := adapter.ValidateToken(ctx, "token")

	// 4. Assertions
	if err != nil {
		t.Fatalf("Expected success, but got error: %v", err)
	}

	expectedUID := "123e4567-e89b-12d3-a456-426614174000"
	if uid != expectedUID {
		t.Errorf("Extracted UID was incorrect. Expected %s, got %s", expectedUID, uid)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// 1. Configure the Mock Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate Keycloak rejecting the token
		// Return an HTTP 401 Unauthorized status
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	// 2. Instantiate the adapter using the mock server URL
	adapter := NewKeycloakAdapter(mockServer.URL, "weiclothe", "dummy-client", "dummy-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute the function with a invalid token
	uid, err := adapter.ValidateToken(ctx, "invalid_token")

	// 4. Assertions
	if err == nil {
		t.Fatalf("Expected an error for an invalid token, but the function returned success")
	}

	if uid != "" {
		t.Errorf("Expected an empty UID for invalid token, but got: %s", uid)
	}
}

func TestLoginUser_Success(t *testing.T) {
	// 1. Configure the Mock Server for Login
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate the adapter is sending the data as a form (Keycloak standard)
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Simulate Keycloak validating the password and returning a JWT Token
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"access_token": "mock_jwt_token", "expires_in": 300}`)
	}))
	defer mockServer.Close()

	// 2. Instantiate the adapter with the mock URL
	adapter := NewKeycloakAdapter(mockServer.URL, "weiclothe", "dummy-client", "dummy-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute the Login with fake data
	token, err := adapter.LoginUser(ctx, "test@weiclothe.com", "password123")

	// 4. Asserts: Verify there is no error and the token matches
	if err != nil {
		t.Fatalf("Expected success, but got error: %v", err)
	}

	expectedToken := "mock_jwt_token"
	if token != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token)
	}
}

func TestRegisterUser_Success(t *testing.T) {
	// 1. Configure the Mock Server to handle TWO different requests
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//  The adapter asks for the Admin Token first
		if strings.Contains(r.URL.Path, "token") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"access_token": "fake_admin_token_123"}`)
			return
		}

		//  The adapter actually registers the user
		// Simulate the Location header returned by Keycloak
		fakeLocation := "http://localhost:9090/admin/realms/weiclothe/users/987f6543-e21b-34c5-b678-1234567890ab"
		w.Header().Set("Location", fakeLocation)
		w.WriteHeader(http.StatusCreated)
	}))
	defer mockServer.Close()

	// 2. Instantiate the adapter
	adapter := NewKeycloakAdapter(mockServer.URL, "weiclothe", "dummy-client", "dummy-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 3. Execute the Registration
	uid, err := adapter.RegisterUser(ctx, "newuser", "new@weiclothe.com", "pass123", "John", "Doe")

	// 4. Asserts
	if err != nil {
		t.Fatalf("Expected success, but got error: %v", err)
	}

	expectedUID := "987f6543-e21b-34c5-b678-1234567890ab"
	if uid != expectedUID {
		t.Errorf("Expected UID %s, got %s", expectedUID, uid)
	}
}

func TestDeleteUser_Success(t *testing.T) {
	//1. Configure the mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Ask for the admin token
		if strings.Contains(r.URL.Path, "token") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"access_token": "fake_admin_token_123"}`)
			return
		}

		//Simulate the response returned by keycloak
		w.WriteHeader(http.StatusNoContent)
	}))

	defer mockServer.Close()

	//2. Instantiate the adapter

	adapter := NewKeycloakAdapter(mockServer.URL, "weiclothe", "dummy-client", "dummy-secret")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	//3. Execute Deletion
	err := adapter.DeleteUser(ctx, "e21b-34c5-b678-1234567890ab")

	//4. Asserts
	if err != nil {
		t.Fatalf("Expected success, but got error: %v", err)
	}
}
