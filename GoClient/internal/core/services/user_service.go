package services

import (
	"context"
	"encoding/json"
	"fmt"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

// UserService orchestrates user operations across Auth, DB and Event logic.
type UserService struct {
	identityProvider ports.IdentityProvider
	userRepository   ports.UserRepository
	eventPublisher   ports.EventPublisher
}

// NewUserService creates a new user service instance with injected dependencies.
func NewUserService(idp ports.IdentityProvider, repo ports.UserRepository, events ports.EventPublisher) *UserService {
	return &UserService{
		identityProvider: idp,
		userRepository:   repo,
		eventPublisher:   events,
	}
}

// RegisterUser handles the entire registration flow
func (s *UserService) RegisterUser(ctx context.Context, input domain.RegisterUserInput) error {

	// 1. Create user in the Identity Provider (Keycloak)
	uid, err := s.identityProvider.RegisterUser(
		ctx,
		input.Nickname,
		input.Email,
		input.Password,
		input.FirstName,
		input.LastName,
	)

	if err != nil {
		return fmt.Errorf("failed to register user in identity provider: %w", err)
	}

	// 2. Map the domain object and save into Database (Postgres)
	userRecord := &domain.User{
		SubKeycloak: uid,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Nickname:    input.Nickname,
		Email:       input.Email,
		DateBirth:   input.DateBirth,
		Gender:      input.Gender,
	}

	err = s.userRepository.CreateUser(ctx, userRecord)
	if err != nil {
		// TODO: Implement Keycloak rollback for now as per design decision
		return fmt.Errorf("failed directly in database: %w", err)
	}

	// 3. Publish the success event to the Broker (Kafka)
	eventPayload := map[string]string{
		"uid":   uid,
		"email": input.Email,
	}

	payloadBytes, _ := json.Marshal(eventPayload)

	err = s.eventPublisher.Publish(ctx, "user.created", uid, payloadBytes)
	if err != nil {
		fmt.Printf("Warning: user was created but event publication failed: %v", err)
		// We usually don't return an error here because the creation itself was successful.
	}

	return nil
}
