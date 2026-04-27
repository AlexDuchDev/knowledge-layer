package onboarding

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// Sentinel errors for HTTP mapping.
var (
	ErrForbidden       = errors.New("forbidden")
	ErrAlreadyLaunched = errors.New("session already launched")
)

// IsNotFound reports pgx.ErrNoRows.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// IsForbidden reports ErrForbidden.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsAlreadyLaunched reports ErrAlreadyLaunched.
func IsAlreadyLaunched(err error) bool {
	return errors.Is(err, ErrAlreadyLaunched)
}
