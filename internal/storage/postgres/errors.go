package postgres

import "errors"

// sqlStateError is implemented by PostgreSQL drivers (including pgx's stdlib adapter).
type sqlStateError interface {
	SQLState() string
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr sqlStateError
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
