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

		// 1. Catch the error from the delete attempt in Keycloak
		rollbackErr := s.identityProvider.DeleteUser(ctx, uid)

		// 2. If the deletion failed
		if rollbackErr != nil {
			return fmt.Errorf("CRITICAL: db fail (%v), AND keycloak rollback failed, user orphaned! (%v)", err, rollbackErr)
		}
		// 3. The deletion was successful
		return fmt.Errorf("failed directly in database, successfully rolled back keycloak: %w", err)
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

func (s *UserService) UpdateUser(ctx context.Context, uid string, input domain.UpdateUserInput) error {
	//1. Map the domain object
	userRecord := &domain.UpdateUserInput{
		SubKeycloak: uid,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Nickname:    input.Nickname,
		DateBirth:   input.DateBirth,
		Gender:      input.Gender,
	}

	err := s.userRepository.UpdateUser(ctx, userRecord)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// 2. Publish the success event to the Broker (Kafka)
	eventPayload := map[string]string{
		"uid": userRecord.SubKeycloak,
	}

	payloadBytes, _ := json.Marshal(eventPayload)

	err = s.eventPublisher.Publish(ctx, "user.updated", userRecord.SubKeycloak, payloadBytes)
	if err != nil {
		fmt.Printf("Warning: user was updated but event publication failed: %v", err)
	}

	return nil
}
