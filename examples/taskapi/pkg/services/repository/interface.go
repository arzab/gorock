package repository

import (
	"context"
	"errors"

	"github.com/arzab/gorock-kit/rockservices"
)

// ErrNotFound is returned when there is no such row; handlers check it with errors.Is.
var ErrNotFound = errors.New("repository: not found")

// Service stores any bun model. It knows nothing about internal/models:
// the caller passes a pointer to its own struct.
type Service interface {
	rockservices.Service

	Insert(ctx context.Context, model any) error
	GetByID(ctx context.Context, model any, id string) error
}
