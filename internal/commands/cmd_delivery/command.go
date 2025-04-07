package cmd_delivery

import (
	"github.com/arzab/gorock/internal/commands/cmd_delivery/cmd_endpoint"
	"github.com/urfave/cli/v3"
)

var endpointPath string
var endpointMethod string

var Command = &cli.Command{
	Name:      "delivery",
	Usage:     "commands to manage delivery directory and entities like 'endpoints', 'processor', etc",
	UsageText: `service [options] {command}`,
	ArgsUsage: `service [options] {command}`,
	Commands: []*cli.Command{
		cmd_endpoint.Command,
	},
}
