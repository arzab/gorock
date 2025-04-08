package commands

import (
	"github.com/arzab/gorock/internal/commands/cmd_delivery"
	"github.com/arzab/gorock/internal/commands/cmd_internal"
	"github.com/arzab/gorock/internal/middlewares"
	"github.com/urfave/cli/v3"
)

func prepareCommand(cmd *cli.Command) *cli.Command {
	cmd.Before = middlewares.GoModCheckErr
	return cmd
}

var Commands = []*cli.Command{
	prepareCommand(cmd_delivery.Command),
	prepareCommand(cmd_internal.Command),
}
