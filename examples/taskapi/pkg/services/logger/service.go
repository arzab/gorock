package logger

import (
	"context"
	"fmt"

	"github.com/arzab/gorock-catalog/rocklog"
	"github.com/arzab/gorock-kit/rockservices"
)

type Configs struct {
	Level  string `config:",default=info"` // debug | info | warn | error
	Format string `config:",default=json"` // json | text
}

// Service configures the global rocklog logger. Code logs through rocklog's
// package functions: rocklog.Info, rocklog.FromContext(ctx).Error...
type Service interface {
	rockservices.Service
}

type service struct{ cfg Configs }

func NewService(cfg Configs) Service { return &service{cfg: cfg} }

func (s *service) Init(context.Context) error {
	levels := map[string]rocklog.Level{
		"debug": rocklog.LevelDebug, "info": rocklog.LevelInfo,
		"warn": rocklog.LevelWarn, "error": rocklog.LevelError,
	}
	level, ok := levels[s.cfg.Level]
	if !ok {
		return fmt.Errorf("unknown log level %q", s.cfg.Level)
	}
	format := rocklog.FormatJSON
	if s.cfg.Format == "text" {
		format = rocklog.FormatText
	}
	rocklog.Init(rocklog.Config{Level: level, Format: format})
	return nil
}

func (s *service) Stop() error { return nil }
