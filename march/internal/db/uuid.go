package db

import "github.com/google/uuid"

// NewID Returns a UUIDv7 for primary keys
func NewID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
