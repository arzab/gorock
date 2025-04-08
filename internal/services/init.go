package services

import (
	"github.com/arzab/gorock/pkg/configs"
	"github.com/arzab/gorock/internal/services/asd"
	"github.com/arzab/gorock/internal/services/asfasfd"
	"github.com/arzab/gorock/internal/services/asfaf"
	//Imports
)

func Init(path string) error {
	cfg, err := configs.Init[Configs](path)
	if err != nil {
		return err
	}
	cfg = cfg

	varAsd = asd.NewService(cfg.Asd)
	varAsfasfd = asfasfd.NewService(cfg.Asfasfd)
	varAsfaf = asfaf.NewService(cfg.Asfaf)
	//Init

	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	//Stop

	return errors
}
