package cmd_endpoint

import (
	"context"
	"fmt"
	"github.com/arzab/gorock/internal/middlewares"
	"github.com/arzab/gorock/internal/utils"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"os"
	"strings"
)

var (
	endpointName   string
	endpointPath   string
	endpointMethod string
)

func action(ctx context.Context, cmd *cli.Command) error {
	fs := afero.NewOsFs()

	endpointsDirPath := "delivery/endpoints"
	err := utils.CheckDir(fs, endpointsDirPath, true, true)
	if err != nil {
		return err
	}

	err = utils.CheckDir(fs, fmt.Sprintf("%s/%s", endpointsDirPath, endpointName), false, false)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", endpointName)
		}
	}

	exampleDirPath := fmt.Sprintf("template/%s", endpointsDirPath)
	dirEntry, err := afero.ReadDir(fs, exampleDirPath)
	if err != nil {
		return fmt.Errorf("could not read example directory %s: %s", exampleDirPath, err)
	}
	for _, info := range dirEntry {
		if info.IsDir() {
			err = utils.CreateFromDir(
				fs,
				fmt.Sprintf("%s/%s", exampleDirPath, info.Name()),
				fmt.Sprintf("%s/%s", endpointsDirPath, endpointName),
				utils.DataUpdater{
					utils.Replace: []utils.OperationArgs{
						{"{path}", endpointPath},
						{"{method}", strings.ToLower(endpointMethod)},
						{"package example", fmt.Sprintf("package %s", endpointName)},
					},
				},
			)
			if err != nil {
				return fmt.Errorf("could not create example directory %s: %s", exampleDirPath, err)
			}
		} else {
			filePath := fmt.Sprintf("%s/%s", endpointsDirPath, info.Name())
			exists, err := afero.Exists(fs, filePath)
			if err != nil {
				return fmt.Errorf("could not check if %s exists: %s", info.Name(), err)
			}
			dataUpdater := utils.DataUpdater{
				utils.AddBefore: {
					{"//Imports", fmt.Sprintf("\"%s/%s/%s\"", middlewares.ProjectName, endpointsDirPath, endpointName)},
					{"//Endpoints", fmt.Sprintf("result = append(result, %s.Endpoint())", endpointName)},
				},
			}
			if exists {
				err = utils.CreateFromFile(
					fs,
					fmt.Sprintf("%s/%s", endpointsDirPath, info.Name()),
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

func action1(ctx context.Context, cmd *cli.Command) error {
	fs := afero.NewOsFs()

	// Init main files
	endpointsDirPath := "delivery/endpoints"

	err := utils.CheckDir(fs, endpointsDirPath, true, true)
	if err != nil {
		return err
	}

	endpointsFilePath := fmt.Sprintf("%s/endpoints.go", endpointsDirPath)
	exists, err := afero.Exists(fs, endpointsFilePath)
	if err != nil {
		return fmt.Errorf("check existence of %s: %w", endpointsFilePath, err)
	}
	if !exists {
		err = utils.CreateFromFile(fs, fmt.Sprintf("template/%s", endpointsFilePath), endpointsFilePath, nil)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	// create endpoint dir
	endpointDirPath := fmt.Sprintf("%s/%s", endpointsDirPath, endpointName)
	err = utils.CheckDir(fs, endpointDirPath, false, false)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s dir already exists", endpointsDirPath)
		}
		return err
	}
	err = fs.Mkdir(endpointDirPath, 0755)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", endpointDirPath, err)
	}
	// get template endpoint
	exampleDirPath := fmt.Sprintf("template/%s/example", endpointsDirPath)
	dirEntry, err := afero.ReadDir(fs, exampleDirPath)
	if err != nil {
		return fmt.Errorf("read example dir: %w", err)
	}
	// copy files from templates to endpoint
	for _, info := range dirEntry {
		err = utils.CreateFromFile(
			fs,
			fmt.Sprintf("%s/%s", exampleDirPath, info.Name()),
			fmt.Sprintf("%s/%s", endpointDirPath, info.Name()),
			utils.DataUpdater{
				utils.Replace: []utils.OperationArgs{
					{"{path}", endpointPath},
					{"{method}", strings.ToLower(endpointMethod)},
					{"package example", fmt.Sprintf("package %s", endpointName)},
				},
			},
		)
		if err != nil {
			return fmt.Errorf("create example %s: %w", info.Name(), err)
		}
	}

	// add endpoint to function httpEndpoints
	data, err := afero.ReadFile(fs, endpointsFilePath)
	if err != nil {
		return fmt.Errorf("read example %s: %w", endpointsFilePath, err)
	}

	data = utils.DataUpdater{
		utils.AddAfter: []utils.OperationArgs{
			{
				Find: "import (",
				Value: fmt.Sprintf("\"%s/delivery/endpoints/%s\"",
					middlewares.ProjectName,
					endpointName,
				),
			},
		},
		utils.AddBefore: []utils.OperationArgs{
			{
				Find:  "//Endpoints",
				Value: fmt.Sprintf("result = append(result, %s.Endpoint())", endpointName),
			},
		},
	}.Update(data)

	err = afero.WriteFile(fs, endpointsFilePath, data, 0755)
	if err != nil {
		return fmt.Errorf("copy example %s: %w", endpointsFilePath, err)
	}
	return nil
}
