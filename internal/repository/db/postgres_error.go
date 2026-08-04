package db

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
)

type PGErrorClassification int

const (
	NonRetriable PGErrorClassification = iota

	Retriable
)

type PostgresErrorClassifier struct {
}

func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

func (pe *PostgresErrorClassifier) ClassifyRetry(err error) PGErrorClassification {
	if err == nil {
		return NonRetriable
	}

	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return Retriable
	}

	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return ClassifyPgError(pgError)
	}

	return NonRetriable
}

func ClassifyPgError(err *pgconn.PgError) PGErrorClassification {
	if pgerrcode.IsConnectionException(err.Code) {
		return Retriable
	}

	if lo.Contains([]string{
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.AdminShutdown,
		pgerrcode.TooManyConnections,
		pgerrcode.QueryCanceled,
		pgerrcode.IOError,
	}, err.Code) {
		return Retriable
	}

	return NonRetriable
}
