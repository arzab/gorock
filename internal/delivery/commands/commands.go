package commands

import (
	"github.com/arzab/gorock/internal/delivery/middlewares"
	"github.com/urfave/cli/v3"
)

func prepareCommand(cmd *cli.Command) *cli.Command {
	cmd.Before = middlewares.GoModCheckErr
	return cmd
}

var Commands = []*cli.Command{
	prepareCommand(Service),
}
