package services

import (
	// Import
	"github.com/arzab/gorock/pkg/configs"
	// Import
)

func Init(path string) error {
	cfg, err := configs.Init[Configs](path)
	if err != nil {
		return err
	}
	cfg = cfg

	// Init
	// Init

	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	// Stop
	// Stop

	return errors
}
