package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/arzab/gorock-kit/rockconfig"
	"github.com/arzab/gorock-kit/rockengine"
	"github.com/arzab/gorock-kit/rockservices"

	"__MODULE__/cmd/apps"
	"__MODULE__/pkg/services"
)

// Configs aggregates the configs of every layer; configs/configs.yaml mirrors it.
type Configs struct {
	Services services.Configs
	Apps     apps.Configs
}

func main() {
	if err := Exec(); err != nil {
		log.Fatal(err)
	}
}

func Exec() error {
	path := "./configs/configs.yaml"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	cfg, err := rockconfig.InitFromFile[Configs](path)
	if err != nil {
		return fmt.Errorf("init configs: %w", err)
	}

	services.Register(rockservices.Default(), cfg.Services)
	if err := rockservices.Init(context.Background()); err != nil {
		return fmt.Errorf("init services: %w", err)
	}
	defer func() {
		if err := rockservices.Stop(); err != nil {
			log.Print(err)
		}
	}()

	engine := rockengine.NewEngine()
	apps.Register(engine, cfg.Apps)
	return engine.Run()
}
