package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type StyleRepository struct {
	db *pgxpool.Pool
}

func NewStyleRepository(db *pgxpool.Pool) *StyleRepository {
	return &StyleRepository{db: db}
}
