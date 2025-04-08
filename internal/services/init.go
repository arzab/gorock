package services

import (
	"github.com/arzab/gorock/pkg/configs"
	"github.com/arzab/gorock/internal/services/hello"
	//Imports
)

func Init(path string) error {
	cfg, err := configs.Init[Configs](path)
	if err != nil {
		return err
	}
	cfg = cfg

	varHello = hello.NewService(cfg.Hello)
	//Init

	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	//Stop

	return errors
}
