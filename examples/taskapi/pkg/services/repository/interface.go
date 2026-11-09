package repository

import (
	"context"
	"errors"
)

// ErrNotFound is returned when no row matches.
var ErrNotFound = errors.New("repository: not found")

// Service stores any bun model. It knows nothing about internal/models:
// the caller passes a pointer to its own struct.
type Service interface {
	// Init / Stop make the service a rockengine.Resource.
	Init(ctx context.Context) error
	Stop() error

	Ping(ctx context.Context) error
	Insert(ctx context.Context, model any) error
	GetByID(ctx context.Context, model any, id string) error
	Update(ctx context.Context, model any) error
}
