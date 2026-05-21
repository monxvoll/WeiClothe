package postgres

import (
	"context"
	"fmt"

	"weicloth/internal/core/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (pgx *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query :=
		`
	INSERT INTO users (sub_keycloak, first_name, last_name, nickname, email, date_birth, gender)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id
	`
	var id string
	err := pgx.db.QueryRow(ctx, query, user.SubKeycloak, user.FirstName, user.LastName, user.Nickname, user.Email, user.DateBirth, user.Gender).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (pgx *UserRepository) UpdateUser(ctx context.Context, user *domain.UpdateUserInput) error {
	query :=
		`
		UPDATE users SET first_name = $1, last_name = $2, nickname = $3, date_birth = $4, gender = $5, updated_at = CURRENT_TIMESTAMP 
		WHERE sub_keycloak = $6
		`
	_, err := pgx.db.Exec(ctx, query, user.FirstName, user.LastName, user.Nickname, user.DateBirth, user.Gender, user.SubKeycloak)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}
