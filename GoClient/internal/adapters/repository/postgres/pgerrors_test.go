package postgres

import (
	"errors"
	"testing"

	"weicloth/internal/core/apperrors"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapPgErr_CheckViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23514", ConstraintName: "clothes_status_check"}
	mapped := mapPgErr(pgErr)
	if !errors.Is(mapped, apperrors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", mapped)
	}
}

func TestMapPgErr_Other(t *testing.T) {
	orig := errors.New("connection reset")
	if mapPgErr(orig) != orig {
		t.Fatal("expected unchanged error")
	}
}
