package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"weicloth/internal/adapters/iam_keycloak"
	"github.com/joho/godotenv"
)

func main() {
	// 0. Load the .env file from the root directory
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: No .env file found. Falling back to system environment variables.")
	}

	// 1. Keycloak Configuration 
	baseURL := os.Getenv("KEYCLOAK_BASE_URL")
	realm := os.Getenv("KEYCLOAK_REALM")
	clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")

	// Safety check 
	if clientSecret == "" {
		log.Fatal("Fatal Error: KEYCLOAK_CLIENT_SECRET is missing. Check your .env file.")
	}

	adapter := iam_keycloak.NewKeycloakAdapter(baseURL, realm, clientID, clientSecret)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load test user data from environment
	testUsername := "test"
	testEmail := os.Getenv("TEST_USER_EMAIL")
	testPassword := os.Getenv("TEST_USER_PASS")

	// REGISTRATION
	registeredUID, err := adapter.RegisterUser(ctx, testUsername, testEmail, testPassword, "Engineer", "Backend")
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Assuming the user already exists. Skipping to Login...")
	} else {
		fmt.Printf("User successfully created in the database.\n")
		fmt.Printf(" (UID): %s\n", registeredUID)
	}

	// LOGIN
	token, err := adapter.LoginUser(ctx, testEmail, testPassword)
	if err != nil {
		log.Fatalf("Fatal login error: %v", err)
	}

	fmt.Printf("Login successful. Keycloak gave us a JWT Token.\n")
	fmt.Printf("Token : %s...\n", token[:30])

	// TOKEN VALIDATION
	validatedUID, err := adapter.ValidateToken(ctx, token)
	if err != nil {
		log.Fatalf("Fatal error validating token: %v", err)
	}

	fmt.Printf("Token is 100%% valid and active.\n")
	fmt.Printf("ID extracted from token (Sub): %s\n", validatedUID)
}