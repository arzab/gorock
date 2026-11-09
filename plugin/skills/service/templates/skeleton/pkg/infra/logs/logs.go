package logs

import (
	"fmt"

	"github.com/arzab/gorock-kit/rocklog"
)

// Configs is the file-loadable form of rocklog.Config.
type Configs struct {
	Level  string `config:",default=info"` // debug | info | warn | error
	Format string `config:",default=json"` // json | text
	Caller bool   `config:",omitempty"`
}

// Init configures the global rocklog logger. Every layer then logs through
// rocklog.Info / rocklog.FromContext(ctx) — never through a concrete library.
func Init(cfg Configs) error {
	levels := map[string]rocklog.Level{
		"debug": rocklog.LevelDebug, "info": rocklog.LevelInfo,
		"warn": rocklog.LevelWarn, "error": rocklog.LevelError,
	}
	level, ok := levels[cfg.Level]
	if !ok {
		return fmt.Errorf("unknown log level %q", cfg.Level)
	}

	format := rocklog.FormatJSON
	if cfg.Format == "text" {
		format = rocklog.FormatText
	}

	rocklog.Init(rocklog.Config{Level: level, Format: format, Caller: cfg.Caller})
	return nil
}
