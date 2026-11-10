package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/arzab/gorock-catalog/rockbun"
)

type service struct {
	db rockbun.Service // child service: the repository owns it
}

// NewService only stores the config and creates children. No connections here.
func NewService(cfg Configs) Service {
	return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }

func (s *service) Insert(ctx context.Context, model any) error {
	_, err := s.db.DB().NewInsert().Model(model).Exec(ctx)
	return err
}

func (s *service) GetByID(ctx context.Context, model any, id string) error {
	err := s.db.DB().NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
