package db

// SQLSTATE codes Postgres returns when a write breaks a constraint.
// See https://www.postgresql.org/docs/18/errcodes-appendix.html.
const (
	// foreignKeyViolation: the write references a row that does not exist.
	foreignKeyViolation = "23503"
	// uniqueViolation: the write duplicates a value in a unique constraint.
	uniqueViolation = "23505"
)
