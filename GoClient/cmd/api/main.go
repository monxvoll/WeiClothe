package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"

	keycloakAdapter "weicloth/internal/adapters/iam_keycloak"

	"weicloth/internal/adapters/handler"

	kafkaAdapter "weicloth/internal/adapters/event_publisher/kafka"

	postgresAdapter "weicloth/internal/adapters/repository/postgres"
	services "weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
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

	keycloak := keycloakAdapter.NewKeycloakAdapter(baseURL, realm, clientID, clientSecret)

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
	clotheService := services.NewClotheService(clotheRepo, producer)

	_ = styleRepo

	/*
		Integration tests in-process (uncomment to exercise Keycloak, Postgres, Kafka flows).
		Comment out the HTTP block below if you run this.

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		testEmail := os.Getenv("TEST_USER_EMAIL")
		testPassword := os.Getenv("TEST_USER_PASS")

		fmt.Println(" User Creation Flow..")
		input := domain.RegisterUserInput{
			FirstName: "Integration",
			LastName:  "TestUser",
			Nickname:  "integ_test1",
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

		token, loginErr := keycloak.LoginUser(ctx, testEmail, testPassword)
		if loginErr != nil {
			log.Fatalf("Fatal: Could not login to perform update: %v", loginErr)
		}

		uid, valErr := keycloak.ValidateToken(ctx, token)
		if valErr != nil {
			log.Fatalf("Fatal: Could not decode Token: %v", valErr)
		}

		fmt.Printf("UID successfully intercepted: %s\n", uid)
		updateInput := domain.UpdateUserInput{
			FirstName: "My New Name",
			LastName:  "My New Lastname",
			Nickname:  "superHacker777",
			DateBirth: time.Date(1999, 9, 9, 0, 0, 0, 0, time.UTC),
			Gender:    "Apache Helicopter",
		}

		updateErr := userService.UpdateUser(ctx, uid, updateInput)
		if updateErr != nil {
			log.Fatalf("Fatal: Update failed: %v", updateErr)
		}
		fmt.Println("SUCCESS: The user has been updated in Postgres and announced in Kafka!")

		clothe := domain.Garment{
			UserID:      uid,
			ImageURL:    "s3://bucket/img.jpg",
			GarmentType: "shirt",
			Source:      "ai",
			Status:      "queued",
		}
		err = clotheService.RegisterClothe(ctx, &clothe)
		if err != nil {
			log.Fatalf("Fatal: Register clothe failed: %v", err)
		}
		fmt.Println("SUCCESS: The clothe has been registered in Postgres and announced in Kafka!")
	*/

	httpHandler := handler.NewHTTPHandler(userService, clotheService)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"}, // La URL exacta de tu Angular
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	httpHandler.RegisterRoutes(r, keycloak)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("HTTP listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
