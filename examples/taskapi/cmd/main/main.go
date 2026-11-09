package main

import (
	"fmt"
	"log"
	"os"

	"github.com/arzab/gorock-kit/rockconfig"
	"github.com/arzab/gorock-kit/rockengine"

	"github.com/yourname/taskapi/cmd/apps"
	"github.com/yourname/taskapi/pkg/infra"
	"github.com/yourname/taskapi/pkg/services"
)

// Configs aggregates the configs of every layer. The config file mirrors it.
type Configs struct {
	Infra    infra.Configs
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

	// infra first: everything after it may log.
	if err := infra.Init(cfg.Infra); err != nil {
		return fmt.Errorf("init infra: %w", err)
	}

	engine := rockengine.NewEngine()
	services.Register(engine, cfg.Services) // resources: init first, stop last
	apps.Register(engine, cfg.Apps)         // apps: start after resources, stop in reverse order

	return engine.Run() // blocks until SIGINT/SIGTERM or an app fails
}
