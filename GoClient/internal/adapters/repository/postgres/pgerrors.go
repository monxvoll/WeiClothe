package postgres

import (
	"errors"
	"fmt"

	"weicloth/internal/core/apperrors"

	"github.com/jackc/pgx/v5/pgconn"
)

func mapPgErr(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" {
		return fmt.Errorf("%w: %s", apperrors.ErrInvalidInput, pgErr.ConstraintName)
	}
	return err
}
