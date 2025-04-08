package services

import (
	"github.com/arzab/gorock/pkg/configs"
	//Imports
)

func Init(path string) error {
	cfg, err := configs.Init[Configs](path)
	if err != nil {
		return err
	}
	cfg = cfg

	//Init

	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	//Stop

	return errors
}
