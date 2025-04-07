package cmd_endpoint

import (
	"context"
	"fmt"
	"github.com/arzab/gorock/internal/middlewares"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
	"os"
	"regexp"
	"strings"
)

var (
	endpointName   string
	endpointPath   string
	endpointMethod string
)

func checkDir(fs afero.Fs, path string, mustExists, force bool) error {
	exists, err := afero.DirExists(fs, path)
	if err != nil {
		return fmt.Errorf("check existence of %s dir: %w", path, err)
	}
	if mustExists {
		if !exists {
			if force {
				return fs.MkdirAll(path, 0755)
			} else {
				return os.ErrNotExist
			}
		}
		return nil
	} else {
		if exists {
			if force {
				return fs.Remove(path)
			} else {
				return os.ErrExist
			}
		}
		return nil
	}
}

func createFromFile(fs afero.Fs, src, dest string, replace map[string]string) error {
	exists, err := afero.Exists(fs, src)
	if err != nil {
		return fmt.Errorf("check existence of %s src: %w", src, err)

	}
	if !exists {
		return fmt.Errorf("%s src does not exist", src)
	}
	data, err := afero.ReadFile(fs, src)
	if err != nil {
		return fmt.Errorf("read file %s: %w", src, err)
	}

	newData := string(data)
	for key, value := range replace {
		newData = strings.ReplaceAll(newData, key, value)
	}
	data = []byte(newData)

	err = afero.WriteFile(fs, dest, data, 0755)
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", src, dest, err)
	}
	return nil
}

func action(ctx context.Context, cmd *cli.Command) error {
	fs := afero.NewOsFs()

	// Init main files
	endpointsDirPath := "delivery/endpoints"

	err := checkDir(fs, endpointsDirPath, true, true)
	if err != nil {
		return err
	}

	endpointsFilePath := fmt.Sprintf("%s/endpoints.go", endpointsDirPath)
	exists, err := afero.Exists(fs, endpointsFilePath)
	if err != nil {
		return fmt.Errorf("check existence of %s: %w", endpointsFilePath, err)
	}
	if !exists {
		err = createFromFile(fs, fmt.Sprintf("template/%s", endpointsFilePath), endpointsFilePath, nil)
		if err != nil && !os.IsExist(err) {
			return err
		}

	}

	// create endpoint dir
	endpointDirPath := fmt.Sprintf("%s/%s", endpointsDirPath, endpointName)
	err = checkDir(fs, endpointDirPath, false, false)
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
		err = createFromFile(
			fs,
			fmt.Sprintf("%s/%s", exampleDirPath, info.Name()),
			fmt.Sprintf("%s/%s", endpointDirPath, info.Name()),
			map[string]string{
				"{path}":          endpointPath,
				"{method}":        strings.ToLower(endpointMethod),
				"package example": fmt.Sprintf("package %s", endpointName),
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
	newData := string(data)

	regexImport := regexp.MustCompile(`import \((\n|[^)])+`)
	imports := regexImport.FindString(newData)
	imports = fmt.Sprintf("%s\t%s\n", imports, fmt.Sprintf("\"%s/delivery/endpoints/%s\"", middlewares.ProjectName, endpointName))
	newData = regexImport.ReplaceAllString(newData, imports)

	regexEndpoints := regexp.MustCompile(`//Endpoints(\n|.)+//Endpoints`)
	endpoints := regexEndpoints.FindString(newData)
	endpoints = strings.TrimRight(endpoints, "//Endpoints")
	endpoints = fmt.Sprintf("%s%s\n\t//Endpoints", endpoints, fmt.Sprintf("result = append(result, %s.Endpoint())", endpointName))
	newData = regexEndpoints.ReplaceAllString(newData, endpoints)

	err = afero.WriteFile(fs, endpointsFilePath, []byte(newData), 0755)
	if err != nil {
		return fmt.Errorf("copy example %s: %w", endpointsFilePath, err)
	}
	return nil
}
