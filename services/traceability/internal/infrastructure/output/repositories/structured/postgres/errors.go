package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"rutadeorigen/traceability/internal/platform/exception"
)

const uniqueViolation = "23505"

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == constraint
}

func internalError(technicalMessage string, cause error) error {
	return exception.New(exception.CodeInternal).
		WithTechnical(technicalMessage).
		WithCause(cause)
}
