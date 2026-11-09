package infra

import (
	"fmt"

	"__MODULE__/pkg/infra/logs"
)

type Configs struct {
	Logs logs.Configs `config:",omitempty"`
}

// Init configures project-wide primitives. Called first, before the engine.
func Init(cfg Configs) error {
	if err := logs.Init(cfg.Logs); err != nil {
		return fmt.Errorf("init logs: %w", err)
	}
	return nil
}
