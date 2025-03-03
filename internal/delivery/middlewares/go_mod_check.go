package middlewares

import (
	"bufio"
	"context"
	"fmt"
	"github.com/urfave/cli/v3"
	"os"
	"strings"
)

var ProjectName string

func GoModCheckErr(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	file, err := os.OpenFile("go.mod", os.O_RDONLY, os.ModePerm)
	if err != nil {
		return ctx, fmt.Errorf("go.mod not found, you have to run this command only in go project directory")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	text := scanner.Text()
	ProjectName = strings.Split(text, " ")[1]

	return ctx, nil
}
