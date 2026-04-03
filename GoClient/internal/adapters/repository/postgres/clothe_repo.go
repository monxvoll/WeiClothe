package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ClotheRepository struct {
	db *pgxpool.Pool
}

func NewClotheRepository(db *pgxpool.Pool) *ClotheRepository {
	return &ClotheRepository{db: db}
}

func (pgx *ClotheRepository) IsAliveClothe() {
	fmt.Println("ClotheRepository is alive")
}
