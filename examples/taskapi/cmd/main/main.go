package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/arzab/gorock-kit/rockconfig"

	"github.com/arzab/gorock/examples/taskapi/cmd/apps"
	"github.com/arzab/gorock/examples/taskapi/pkg/services"
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

	ctx := context.Background()
	if err := services.Init(ctx, cfg.Services); err != nil {
		return fmt.Errorf("init services: %w", err) // no App starts
	}
	defer func() {
		if err := services.Stop(); err != nil {
			log.Print(err)
		}
	}()

	return apps.Run(ctx, cfg.Apps)
}
