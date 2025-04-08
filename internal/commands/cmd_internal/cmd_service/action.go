package cmd_service

import (
	"context"
	"fmt"
	"github.com/arzab/gorock/internal/middlewares"
	"github.com/arzab/gorock/internal/utils"
	"github.com/iancoleman/strcase"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"os"
)

func action(ctx context.Context, cmd *cli.Command) error {
	fs := afero.NewOsFs()

	servicesDirPath := "internal/services"
	err := utils.CheckDir(fs, servicesDirPath, true, true)
	if err != nil {
		return err
	}

	err = utils.CheckDir(fs, fmt.Sprintf("%s/%s", servicesDirPath, serviceName), false, false)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", serviceName)
		}
	}

	exampleDirPath := fmt.Sprintf("template/%s", servicesDirPath)
	dirEntry, err := afero.ReadDir(fs, exampleDirPath)
	if err != nil {
		return fmt.Errorf("could not read example directory %s: %s", exampleDirPath, err)
	}
	for _, info := range dirEntry {
		if info.IsDir() {
			err = utils.CreateFromDir(
				fs,
				fmt.Sprintf("%s/%s", exampleDirPath, info.Name()),
				fmt.Sprintf("%s/%s", servicesDirPath, serviceName),
				utils.DataUpdater{
					utils.Replace: []utils.OperationArgs{
						{"package example", fmt.Sprintf("package %s", serviceName)},
					},
				},
			)
			if err != nil {
				return fmt.Errorf("could not create example directory %s: %s", exampleDirPath, err)
			}
		} else {
			filePath := fmt.Sprintf("%s/%s", servicesDirPath, info.Name())
			exists, err := afero.Exists(fs, filePath)
			if err != nil {
				return fmt.Errorf("could not check if %s exists: %s", info.Name(), err)
			}
			dataUpdater := utils.DataUpdater{
				utils.AddBefore: {
					{"//Imports", fmt.Sprintf("\"%s/%s/%s\"", middlewares.ProjectName, servicesDirPath, serviceName)},
					{"//Configs", fmt.Sprintf("%s %s.Configs `json:\"%s\"`", strcase.ToCamel(serviceName), serviceName, strcase.ToSnake(serviceName))},
					{"// Var", fmt.Sprintf("\tvar%s %s.Service", strcase.ToCamel(serviceName), serviceName)},
					{"//Functions", fmt.Sprintf("func %s() %s.Service { return var%s }", strcase.ToCamel(serviceName), serviceName, strcase.ToCamel(serviceName))},
					{"//Init", fmt.Sprintf("var%s = %s.NewService(cfg.%s)", strcase.ToCamel(serviceName), serviceName, strcase.ToCamel(serviceName))},
				},
			}
			if exists {
				err = utils.CreateFromFile(
					fs,
					fmt.Sprintf("%s/%s", servicesDirPath, info.Name()),
					filePath,
					dataUpdater,
				)
				if err != nil {
					return fmt.Errorf("could not create file %s: %s", filePath, err)
				}
			} else {
				err = utils.CreateFromFile(
					fs,
					fmt.Sprintf("%s/%s", exampleDirPath, info.Name()),
					filePath,
					dataUpdater,
				)
				if err != nil {
					return fmt.Errorf("could not create file %s: %s", filePath, err)
				}
			}
		}
	}
	return nil
}
