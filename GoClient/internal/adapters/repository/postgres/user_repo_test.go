package postgres

import (
	"context"
	"fmt"

	"testing"
	"time"
	"weicloth/internal/core/domain"
)

func TestUserRepository_CreateUser(t *testing.T) {

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", "localhost", "5432", "SkinnyMan", "LasEmpanadas777", "weiclothe", "disable")
	postgres, err := NewConnection(context.Background(), dsn)
	if err != nil {
		t.Fatalf("Fatal error connecting to Postgres: %v", err)
	}
	defer postgres.Close()

	pool := NewUserRepository(postgres)

	err = pool.CreateUser(&domain.User{
		SubKeycloak: "test2",
		FirstName:   "Test",
		LastName:    "Test",
		Nickname:    "Test",
		Email:       "tes2@test.com",
		DateBirth:   time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Gender:      "Male",
	})
	if err != nil {
		t.Fatalf("Fatal error creating user: %v", err)
	}

	fmt.Println("User created successfully")
}
