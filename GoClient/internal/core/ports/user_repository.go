package ports

import (
	"context"
	"weicloth/internal/core/domain"
)

type UserRepository interface {
	// Creates a new user record in the database.
	// It takes a user object and returns an error if it fails.
	CreateUser(ctx context.Context, user *domain.User) error

	// Updates basic user information
	UpdateUser(ctx context.Context, user *domain.UpdateUserInput) error
}
