package cmd_internal

import (
	"github.com/arzab/gorock/internal/commands/cmd_internal/cmd_service"
	"github.com/urfave/cli/v3"
)

var endpointPath string
var endpointMethod string

var Command = &cli.Command{
	Name:      "internal",
	Usage:     "commands to manage internal directory and entities like 'services', 'models', etc",
	UsageText: `internal [options] {command}`,
	ArgsUsage: `internal [options] {command}`,
	Commands: []*cli.Command{
		cmd_service.Command,
	},
}
