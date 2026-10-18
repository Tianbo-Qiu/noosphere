package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE codes Postgres returns when a write breaks a constraint.
// See https://www.postgresql.org/docs/18/errcodes-appendix.html.
const (
	// foreignKeyViolation: the write references a row that does not exist.
	foreignKeyViolation = "23503"
	// uniqueViolation: the write duplicates a value in a unique constraint.
	uniqueViolation = "23505"
)

func isConstraintViolation(err error, code, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code && pgErr.ConstraintName == constraint
}
