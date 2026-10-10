package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgExclusionViolation  = "23P01"
)

func pgCode(err error) string {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		return pgErr.Code
	}

	return ""
}
