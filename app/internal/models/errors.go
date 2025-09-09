package models

import "errors"

// Domain specific errors for authentication and authorization.
var (
	ErrNotFound            = errors.New("requested item not found")
	ErrConflict            = errors.New("item already exists or conflict")
	ErrUnauthenticated     = errors.New("authentication required or invalid credentials")
	ErrForbidden           = errors.New("action forbidden")
	ErrBadRequest          = errors.New("bad request")
	ErrValidation          = errors.New("validation failed")
	ErrProfileNameEmpty    = errors.New("profile name cannot be empty")
	ErrProfileNameTooLong  = errors.New("profile name exceeds maximum length")
	ErrCannotDeleteDefault = errors.New("cannot delete default profile")
)
