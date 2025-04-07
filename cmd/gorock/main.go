package main

import (
	"context"
	"fmt"
	"github.com/arzab/gorock/internal/commands"
	"github.com/urfave/cli/v3"
	"os"
)

func main() {
	cmd := &cli.Command{
		Name:      "gorock",
		Aliases:   []string{"gorock"},
		Commands:  commands.Commands,
		ErrWriter: os.Stdout,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
	}
}
