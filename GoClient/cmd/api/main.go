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

	postgresAdapter "weicloth/internal/adapters/repository/postgres"
	"weicloth/internal/core/domain"
	services "weicloth/internal/core/services"

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

	// ── Postgres ──
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))
	postgres, err := postgresAdapter.NewConnection(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Fatal error connecting to Postgres: %v", err)
	}
	defer postgres.Close()

	// ── Repositories ──
	userRepo := postgresAdapter.NewUserRepository(postgres)
	clotheRepo := postgresAdapter.NewClotheRepository(postgres)
	styleRepo := postgresAdapter.NewStyleRepository(postgres)

	// ── Services ──
	userService := services.NewUserService(keycloak, userRepo, producer)

	_ = userService
	_ = clotheRepo
	_ = styleRepo

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test the Architecture Flow
	testEmail := os.Getenv("TEST_USER_EMAIL")
	testPassword := os.Getenv("TEST_USER_PASS")

	fmt.Println(" User Creation Flow..")
	input := domain.RegisterUserInput{
		FirstName: "Integration",
		LastName:  "TestUser",
		Nickname:  "integ_test",
		Email:     testEmail,
		Password:  testPassword,
		DateBirth: time.Date(1995, 5, 20, 0, 0, 0, 0, time.UTC),
		Gender:    "Male",
	}

	err = userService.RegisterUser(ctx, input)
	if err != nil {
		log.Fatalf("Fatal: Flow failed: %v", err)
	}

	fmt.Println("Flow completed successfully! SubKeycloak mapped, Postgres saved, and Kafka event published.")
}
