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

func (pgx *UserRepository) IsAliveUser() {
	fmt.Println("UserRepository is alive")
}

func (pgx *UserRepository) CreateUser(user *domain.User) error {
	query :=
		`
	INSERT INTO users (sub_keycloak, first_name, last_name, nickname, email, date_birth, gender)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id
	`
	var id string
	err := pgx.db.QueryRow(context.Background(), query, user.SubKeycloak, user.FirstName, user.LastName, user.Nickname, user.Email, user.DateBirth, user.Gender).Scan(&id)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}
