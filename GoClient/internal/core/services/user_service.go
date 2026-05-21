package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/ports"
)

// UserService orchestrates user operations across Auth, DB and Event logic.
type UserService struct {
	identityProvider ports.IdentityProvider
	userRepository   ports.UserRepository
	eventPublisher   ports.EventPublisher
	log              *slog.Logger
}

// NewUserService creates a new user service instance with injected dependencies.
func NewUserService(idp ports.IdentityProvider, repo ports.UserRepository, events ports.EventPublisher, logger *slog.Logger) *UserService {
	return &UserService{
		identityProvider: idp,
		userRepository:   repo,
		eventPublisher:   events,
		log:              logger.With("service", "user"),
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

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.log.Error("marshal user.created event", "uid", uid, "err", err)
		return nil
	}

	if err = s.eventPublisher.Publish(ctx, "user.created", uid, payloadBytes); err != nil {
		s.log.Warn("event publish failed", "event", "user.created", "uid", uid, "err", err)
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

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.log.Error("marshal user.updated event", "uid", userRecord.SubKeycloak, "err", err)
		return nil
	}

	if err = s.eventPublisher.Publish(ctx, "user.updated", userRecord.SubKeycloak, payloadBytes); err != nil {
		s.log.Warn("event publish failed", "event", "user.updated", "uid", userRecord.SubKeycloak, "err", err)
	}

	return nil
}

// LoginUser authenticates a user against the Identity Provider (Keycloak)
// and returns the JWT token (or an error).
func (s *UserService) LoginUser(ctx context.Context, input domain.LoginInput) (string, error) {

	//1. Ask Keycloak to generate the token
	token, err := s.identityProvider.LoginUser(ctx, input.Email, input.Password)
	if err != nil {
		return "", fmt.Errorf("login failed: %w", err)
	}

	// Validate token and get UID
	uid, err := s.identityProvider.ValidateToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("token validation failed after login: %w", err)
	}

	//3. Build the event payload
	eventPayload := map[string]string{
		"uid":       uid,
		"email":     input.Email,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payloadBytes, marshalErr := json.Marshal(eventPayload)
	if marshalErr != nil {
		s.log.Error("marshal user.logged_in event", "uid", uid, "err", marshalErr)
		return token, nil
	}

	//4. Publish the success event to the Broker (Kafka)
	if err := s.eventPublisher.Publish(ctx, "user.logged_in", uid, payloadBytes); err != nil {
		s.log.Warn("event publish failed", "event", "user.logged_in", "uid", uid, "err", err)
	}

	return token, nil
}
