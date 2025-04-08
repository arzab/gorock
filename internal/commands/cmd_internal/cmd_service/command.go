package cmd_service

//TODO: Меня тут все бесит, надо переделать
import (
	"github.com/urfave/cli/v3"
)

var serviceName string

var Command = &cli.Command{
	Name:      "service",
	Usage:     "create service directory and files",
	ArgsUsage: `{name}`,
	UsageText: `service [options] {name}`,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:        "name",
			UsageText:   "name of the service",
			Destination: &serviceName,
			Min:         1,
			Max:         1,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Action: action,
}
