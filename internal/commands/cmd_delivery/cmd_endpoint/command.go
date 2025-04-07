package cmd_endpoint

import "github.com/urfave/cli/v3"

var Command = &cli.Command{
	Name:      "endpoint",
	Usage:     "create endpoint directory and files in delivery layer",
	ArgsUsage: `service [options] {method} {path}`,
	UsageText: `service [options] {method} {path}`,
	Arguments: []cli.Argument{
		&cli.StringArg{
			Name:        "name",
			UsageText:   "name of endpoint directory",
			Destination: &endpointName,
			Min:         1,
			Max:         1,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringArg{
			Name:        "method",
			UsageText:   "method of endpoint, GET|POST|PUT|DELETE",
			Destination: &endpointMethod,
			Min:         1,
			Max:         1,
			Config:      cli.StringConfig{TrimSpace: true},
		},
		&cli.StringArg{
			Name:        "endpoint_path",
			UsageText:   "path of endpoint, should start with '/'",
			Destination: &endpointPath,
			Min:         1,
			Max:         1,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	},
	Action: action,
}
