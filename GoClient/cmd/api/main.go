package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"weicloth/internal/adapters/iam_keycloak"

	kafkaAdapter "weicloth/internal/adapters/event_publisher/kafka"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: No .env file found. Falling back to system environment variables.")
	}

	// ── Keycloak ──
	baseURL := os.Getenv("KEYCLOAK_BASE_URL")
	realm := os.Getenv("KEYCLOAK_REALM")
	clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
	clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")

	if clientSecret == "" {
		log.Fatal("Fatal Error: KEYCLOAK_CLIENT_SECRET is missing. Check your .env file.")
	}

	keycloak := iam_keycloak.NewKeycloakAdapter(baseURL, realm, clientID, clientSecret)

	// ── Kafka ──
	brokersRaw := os.Getenv("KAFKA_BROKERS")
	if brokersRaw == "" {
		log.Fatal("Fatal Error: KAFKA_BROKERS is missing. Check your .env file.")
	}
	brokers := strings.Split(brokersRaw, ",")

	producer := kafkaAdapter.NewProducer(kafkaAdapter.DefaultProducerConfig(brokers))
	defer producer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ── Keycloak: register + login + validate ──
	testUsername := "test"
	testEmail := os.Getenv("TEST_USER_EMAIL")
	testPassword := os.Getenv("TEST_USER_PASS")

	registeredUID, err := keycloak.RegisterUser(ctx, testUsername, testEmail, testPassword, "Engineer", "Backend")
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Assuming the user already exists. Skipping to Login...")
	} else {
		fmt.Printf("User successfully created in the database.\n")
		fmt.Printf(" (UID): %s\n", registeredUID)
	}

	token, err := keycloak.LoginUser(ctx, testEmail, testPassword)
	if err != nil {
		log.Fatalf("Fatal login error: %v", err)
	}

	fmt.Printf("Login successful. Keycloak gave us a JWT Token.\n")
	fmt.Printf("Token : %s...\n", token[:30])

	validatedUID, err := keycloak.ValidateToken(ctx, token)
	if err != nil {
		log.Fatalf("Fatal error validating token: %v", err)
	}

	fmt.Printf("Token is 100%% valid and active.\n")
	fmt.Printf("ID extracted from token (Sub): %s\n", validatedUID)

	// ── Kafka: publish test event ──
	payload := []byte(fmt.Sprintf(`{"uid":"%s","action":"login","ts":%d}`, validatedUID, time.Now().Unix()))
	if err := producer.Publish(ctx, "user.login", validatedUID, payload); err != nil {
		log.Fatalf("Failed to publish event to Kafka: %v", err)
	}
	fmt.Println("Event published to Kafka [topic=user.login]")
}