package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/arzab/gorock-kit/rockbun"
)

type service struct {
	db rockbun.Service // child service from gorock-kit
}

// NewService only stores config and creates child services — no connections.
func NewService(cfg Configs) Service {
	return &service{db: rockbun.NewService(cfg.Postgres)}
}

func (s *service) Init(ctx context.Context) error { return s.db.Init(ctx) }
func (s *service) Stop() error                    { return s.db.Stop() }

func (s *service) Ping(ctx context.Context) error { return s.db.Ping(ctx) }

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

func (s *service) Update(ctx context.Context, model any) error {
	res, err := s.db.DB().NewUpdate().Model(model).WherePK().Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
