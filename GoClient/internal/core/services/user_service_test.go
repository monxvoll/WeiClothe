package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

// --- Mock Identity Provider ---
type MockIdentityProvider struct {
	ShouldFail      bool
	DeleteWasCalled bool
}

func (m *MockIdentityProvider) RegisterUser(ctx context.Context, username, email, password, firstName, lastName string) (string, error) {
	if m.ShouldFail {
		return "", errors.New("mock keycloak failed") // Simulate Keycloak error
	}
	return "mock-uuid-keycloak", nil
}

func (m *MockIdentityProvider) LoginUser(ctx context.Context, email, password string) (string, error) {
	return "token", nil
}

func (m *MockIdentityProvider) ValidateToken(ctx context.Context, token string) (string, error) {
	return "mock-uuid-keycloak", nil
}

func (m *MockIdentityProvider) DeleteUser(ctx context.Context, uid string) error {
	m.DeleteWasCalled = true
	if m.ShouldFail {
		return errors.New("mock deletion failed")
	}
	return nil
}

// --- Mock Repository ---
type MockUserRepository struct {
	ShouldFail bool
	SavedUser  *domain.User
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if m.ShouldFail {
		return errors.New("mock postgres failed") // Simulate Postgres error
	}
	m.SavedUser = user
	return nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user *domain.UpdateUserInput) error {
	return nil //Simulate the update
}

// --- Mock Event Publisher ---
type MockEventPublisher struct {
	PublishedTopic string
	PublishedKey   string
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	m.PublishedTopic = topic
	m.PublishedKey = key
	return nil
}

func (m *MockEventPublisher) PublishBatch(ctx context.Context, topic string, messages []ports.Message) error {
	return nil
}

func (m *MockEventPublisher) Close() error { return nil }

// The Integration Test
func TestUserService_RegisterUser_HappyPath(t *testing.T) {
	// 1. Prepare mocks
	mockIdp := &MockIdentityProvider{ShouldFail: false}
	mockRepo := &MockUserRepository{ShouldFail: false}
	mockPub := &MockEventPublisher{}

	// 2. Initialize Service
	userService := NewUserService(mockIdp, mockRepo, mockPub)

	// 3. Prepare Input
	input := domain.RegisterUserInput{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Email:     "john@doe.com",
		Password:  "secret",
		DateBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Gender:    "Male",
	}

	// 4. Execute Service
	err := userService.RegisterUser(context.Background(), input)

	// 5. Assertions
	if err != nil {
		t.Fatalf("Expected strictly no error on happy path, got: %v", err)
	}

	if mockRepo.SavedUser == nil {
		t.Fatal("Expected user to be saved in postgres, but it was nil")
	}

	if mockRepo.SavedUser.SubKeycloak != "mock-uuid-keycloak" {
		t.Errorf("Expected SubKeycloak to be 'mock-uuid-keycloak', got %s", mockRepo.SavedUser.SubKeycloak)
	}

	if mockPub.PublishedTopic != "user.created" {
		t.Errorf("Expected event topic 'user.created', got %s", mockPub.PublishedTopic)
	}
}

// Rollback Test
func TestUserService_RegisterUser_Rollback(t *testing.T) {
	// 1. Prepare mocks
	mockIdp := &MockIdentityProvider{ShouldFail: false}
	mockRepo := &MockUserRepository{ShouldFail: true}
	mockPub := &MockEventPublisher{}

	// 2. Initialize Service
	userService := NewUserService(mockIdp, mockRepo, mockPub)

	// 3. Prepare Input
	input := domain.RegisterUserInput{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Email:     "john@doe.com",
		Password:  "secret",
		DateBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Gender:    "Male",
	}

	// 4. Execute Service
	err := userService.RegisterUser(context.Background(), input)

	// 5. Assertions
	if err == nil {
		t.Fatalf("Expected an error due to postgres fail, but got nil")
	}

	if mockIdp.DeleteWasCalled == false {
		t.Fatalf("Expected IdentityProvider.DeleteUser to be called for rollback, but it was not called")
	}
}

// Update User Test
func TestUserService_UpdateUser_HappyPath(t *testing.T) {
	// 1. Prepare mocks
	mockIdp := &MockIdentityProvider{ShouldFail: false}
	mockRepo := &MockUserRepository{ShouldFail: false}
	mockPub := &MockEventPublisher{}

	// 2. Initialize Service
	userService := NewUserService(mockIdp, mockRepo, mockPub)

	// 3. Prepare Input
	input := domain.UpdateUserInput{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		DateBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Gender:    "Male",
	}

	// 4. Execute Service
	err := userService.UpdateUser(context.Background(), "mock-uuid-keycloak", input)

	// 5. Assertions
	if err != nil {
		t.Fatalf("Expected strictly no error on happy path, got: %v", err)
	}

}
