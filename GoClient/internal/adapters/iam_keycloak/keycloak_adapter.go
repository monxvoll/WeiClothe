package iam_keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"weicloth/internal/core/apperrors"
)

// KeycloakAdapter holds the necessary configuration and HTTP client
// required to communicate with the external Keycloak server.
type KeycloakAdapter struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

// KeycloakTokenResponse is a private struct
// to easily extract the JSON data returned by the Keycloak server.
type keycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// keycloakUserRequest represents the JSON structure Keycloak expects
// when creating a new user via the Admin API.
type keycloakUserRequest struct {
	Username      string               `json:"username"`
	Email         string               `json:"email"`
	FirstName     string               `json:"firstName"`
	LastName      string               `json:"lastName"`
	Enabled       bool                 `json:"enabled"`
	EmailVerified bool                 `json:"emailVerified"`
	Credentials   []keycloakCredential `json:"credentials"`
}

type keycloakCredential struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// keycloakUserInfoResponse represents the JSON returned by the UserInfo endpoint.
type keycloakUserInfoResponse struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// NewKeycloakAdapter initializes and returns a new instance of KeycloakAdapter.
// It receives the environment variables needed
// to establish a connection with the Keycloak Identity Provider.
func NewKeycloakAdapter(baseURL, realm, clientID, clientSecret string, timeout time.Duration) *KeycloakAdapter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &KeycloakAdapter{
		BaseURL:      baseURL,
		Realm:        realm,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   &http.Client{Timeout: timeout},
	}
}

// getAdminToken requests a special token using the Client Credentials flow.
// This token allows Go API to perform administrative tasks like creating users.
func (k *KeycloakAdapter) getAdminToken(ctx context.Context) (string, error) {
	// 1. Build the exact Keycloak URL to request the token
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", k.BaseURL, k.Realm)

	// 2. Prepare the login data using client credentials )
	formData := url.Values{}
	formData.Set("client_id", k.ClientID)
	formData.Set("client_secret", k.ClientSecret)
	formData.Set("grant_type", "client_credentials")

	// 3. Create the HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}

	// 4. Tell Keycloak we are sending web form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 5. Send the request over the internet
	res, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	// 6. Close the network connection to save memory
	defer res.Body.Close()

	// 7. Read the JSON response and put it into our struct
	var tokenRes keycloakTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return "", err
	}

	// 8. Check if Keycloak rejected our admin login
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("admin auth failed (%d): %s", res.StatusCode, tokenRes.Error)
	}

	// 9. Success! Return the admin access token
	return tokenRes.AccessToken, nil
}

// CONTRACT IMPLEMENTATION

// RegisterUser creates a new user in Keycloak and returns the unique User ID.
func (k *KeycloakAdapter) RegisterUser(ctx context.Context, username, email, password, firstName, lastName string) (string, error) {
	// 1. Get the admin token
	adminToken, err := k.getAdminToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get admin token for registration: %w", err)
	}

	// 2. Prepare the Keycloak-specific User JSON
	userBody := keycloakUserRequest{
		Username:      username,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		Enabled:       true,
		EmailVerified: true,
		Credentials: []keycloakCredential{
			{
				Type:      "password",
				Value:     password,
				Temporary: false,
			},
		},
	}

	jsonBody, err := json.Marshal(userBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal user body: %w", err)
	}
	endpoint := fmt.Sprintf("%s/admin/realms/%s/users", k.BaseURL, k.Realm)

	// 3. Create the HTTP POST request to the Admin API
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	// 4. Set Headers: Authentication + Content Type
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	// 5. Execute the creation
	res, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error during registration: %w", err)
	}
	defer res.Body.Close()

	// 6. Keycloak returns 201 Created on success
	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("keycloak registration failed (%d): %s", res.StatusCode, string(body))
	}

	// 7. Extract the User ID from the "Location" header
	location := res.Header.Get("Location")
	segments := strings.Split(location, "/")
	userID := segments[len(segments)-1]

	return userID, nil
}

// LoginUser authenticates a user and retrieves a JWT access token from Keycloak.
func (k *KeycloakAdapter) LoginUser(ctx context.Context, email string, password string) (string, error) {
	// 1. Build the exact Keycloak endpoint URL for requesting tokens
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", k.BaseURL, k.Realm)

	// 2. Prepare the form data (Keycloak requires x-www-form-urlencoded for tokens)
	formData := url.Values{}
	formData.Set("client_id", k.ClientID)
	formData.Set("client_secret", k.ClientSecret)
	formData.Set("grant_type", "password")
	formData.Set("username", email)
	formData.Set("password", password)
	formData.Set("scope", "openid")

	// 3. Create the HTTP request WITH CONTEXT (ctx) to prevent memory leaks
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}

	// 4. Set the mandatory header so Keycloak understands the data format
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 5. Execute the request over the network
	res, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request to Keycloak: %w", err)
	}
	defer res.Body.Close()

	// 6. Read the raw bytes returned by Keycloak
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// 7. Parse the JSON bytes into our Go struct
	var tokenRes keycloakTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenRes); err != nil {
		return "", fmt.Errorf("failed to parse Keycloak response: %w", err)
	}

	// 8. Check if Keycloak rejected the login
	if res.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("%w: %s", apperrors.ErrInvalidCredentials, tokenRes.ErrorDesc)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("keycloak error (%d): %s - %s", res.StatusCode, tokenRes.Error, tokenRes.ErrorDesc)
	}

	// 9.  Return the JWT access token
	return tokenRes.AccessToken, nil
}

// ValidateToken receives a JWT string, sends it to Keycloak's userinfo endpoint,
// and returns the User ID (uid) if the token is valid and active.
func (k *KeycloakAdapter) ValidateToken(ctx context.Context, token string) (string, error) {
	// 1. Build the UserInfo endpoint URL
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", k.BaseURL, k.Realm)

	// 2. Create the GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create validation request: %w", err)
	}

	// 3. Put the token in the Authorization header
	req.Header.Set("Authorization", "Bearer "+token)

	// 4. Send the request
	res, err := k.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error during token validation: %w", err)
	}
	defer res.Body.Close()

	// 5. If the token is fake, expired, or manipulated, Keycloak returns 401 Unauthorized
	if res.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("%w", apperrors.ErrUnauthorized)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid or expired token: keycloak returned status %d", res.StatusCode)
	}

	// 6. If it's valid, decode the JSON to get the User ID
	var userInfo keycloakUserInfoResponse
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return "", fmt.Errorf("failed to decode user info: %w", err)
	}

	// 7. Return the unique User ID (Sub)
	return userInfo.Sub, nil
}

func (k *KeycloakAdapter) DeleteUser(ctx context.Context, uid string) (err error) {
	//1. Get admin token
	adminToken, err := k.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token for delete: %w", err)
	}

	//2. Prepare the endpoint URL for deleting a user
	endpoint := fmt.Sprintf("%s/admin/realms/%s/users/%s", k.BaseURL, k.Realm, uid)

	//3. Create the HTTP DELETE request
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	//4. Set headers
	req.Header.Set("Authorization", "Bearer "+adminToken)

	//5. Execute deletion
	res, err := k.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("network error during registration: %w", err)
	}
	defer res.Body.Close()

	//6. Return nil if deletion was successful
	if res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("keycloak deletion failed (%d): %s", res.StatusCode, res.Status)
	}

	return nil
}
