package commands

import (
	"context"
	"fmt"
	"github.com/arzab/gorock/internal/delivery/middlewares"
	"github.com/iancoleman/strcase"
	"github.com/urfave/cli/v3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var serviceName string

var serviceFilesMap = map[string]string{
	"configs.go": `package %s

type Configs struct {
}
`,
	"interface.go": `package %s

type Service interface {
}
`,
	"service.go": `package %s

type service struct {
	configs Configs
}

func NewService(configs Configs) Service {
	return &service{
		configs: configs,
	}
}
`,
}

var mainFilesMap = map[string]string{
	"configs.go": `package services

// Import start
%v
// Import end


type Configs struct {
// Configs start
	%s
// Configs end
}
%s
`,
	"init.go": `package services

// Import start
%v
// Import end

func Init(configs Configs) error {
	// Init start
	%s
	// Init end
	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	// Stop start
	%s
	// Stop end

	return errors
}`,
	"services.go": `package services

// Import start
%v
// Import end

// Services start
%s
// Services end

%s
`,
}

var Service = &cli.Command{
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

func action(ctx context.Context, cmd *cli.Command) error {
	name := serviceName

	name = strings.ToLower(name)
	name = strcase.ToSnake(name)

	// Создание базовой папки
	mainDirPath := "internal/services"
	err := os.MkdirAll(mainDirPath, 0755)
	if err != nil {
		return fmt.Errorf("create service directory: %w", err)
	}

	// Создание папки
	serviceDirPath := fmt.Sprintf("%s/%s", mainDirPath, name)
	err = os.Mkdir(serviceDirPath, 0755)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("service directory already exists: %s", name)
		}
		return fmt.Errorf("create service directory: %w", err)
	}

	// Создание файлов
	for fileName, content := range serviceFilesMap {
		path := fmt.Sprintf("%s/%s", serviceDirPath, fileName)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return fmt.Errorf("open %s file: %w", fileName, err)
		}
		_, err = file.WriteString(fmt.Sprintf(content, name))
		if err != nil {
			file.Close()
			return fmt.Errorf("write %s file: %w", fileName, err)
		}
		err = file.Close()
		if err != nil {
			return fmt.Errorf("close %s file: %w", fileName, err)
		}
	}

	// Добавление данных в services
	// Создание файлов
	for fileName, content := range mainFilesMap {
		path := fmt.Sprintf("%s/%s", mainDirPath, fileName)

		// Формируем данные для сервиса
		imports := []string{
			fmt.Sprintf(`import "%s/internal/services/%s"`, middlewares.ProjectName, name),
		}

		values1 := make([]string, 0)
		values2 := make([]string, 0)

		//camelName := strcase.ToLowerCamel(name)
		titleName := cases.Title(language.English, cases.NoLower).String(name)

		switch filepath.Base(path) {
		case "configs.go":
			values1 = append(values1, "")
			values1 = append(values1, fmt.Sprintf("\t// %s", titleName))
			values1 = append(values1, fmt.Sprintf("\t%s %s.Configs `json:\"%s\"`", titleName, name, name))

		case "init.go":
			values1 = append(values1, "")
			values1 = append(values1, fmt.Sprintf("\t// %s", titleName))
			values1 = append(values1, fmt.Sprintf("\tvar%s = %s.NewService(configs.%s)", titleName, name, titleName))

		case "services.go":
			values1 = append(values1, "")
			values1 = append(values1, fmt.Sprintf("// %s", titleName))
			values1 = append(values1,
				fmt.Sprintf(`var var%s %s.Service`, titleName, name),
				"",
				fmt.Sprintf(`func %s() %s.Service{return var%s}`, titleName, name, titleName),
			)
		}

		//if data, err := os.ReadFile(path); err == nil {
		//
		//}

		// Собираем данные прошлых сервисов
		data, err := os.ReadFile(path)
		if err == nil {
			// Подбираем данные с import
			importsText := regexp.MustCompile("// Import start(.|\n)+// Import end").FindString(string(data))
			importsText = strings.ReplaceAll(importsText, "// Import start", "")
			importsText = strings.ReplaceAll(importsText, "// Import end", "")
			textData := strings.FieldsFunc(importsText, func(r rune) bool {
				return r == '\n'
			})
			imports = append(imports, textData...)

			switch filepath.Base(path) {
			case "configs.go":
				values1Text := regexp.MustCompile("// Configs start(.|\n)+// Configs end").FindString(string(data))
				values1Text = strings.ReplaceAll(values1Text, "// Configs start", "")
				values1Text = strings.ReplaceAll(values1Text, "// Configs end", "")
				textData = strings.FieldsFunc(values1Text, func(r rune) bool {
					return r == '\n'
				})
				values1 = append(values1, textData...)
			case "init.go":
				values1Text := regexp.MustCompile("// Init start(.|\n)+// Init end").FindString(string(data))
				values1Text = strings.ReplaceAll(values1Text, "// Init start", "")
				values1Text = strings.ReplaceAll(values1Text, "// Init end", "")
				textData = strings.FieldsFunc(values1Text, func(r rune) bool {
					return r == '\n'
				})
				values1 = append(values1, textData...)
			case "services.go":
				values1Text := regexp.MustCompile("// Services start(.|\n)+// Services end").FindString(string(data))
				values1Text = strings.ReplaceAll(values1Text, "// Services start", "")
				values1Text = strings.ReplaceAll(values1Text, "// Services end", "")
				textData = strings.FieldsFunc(values1Text, func(r rune) bool {
					return r == '\n'
				})
				values1 = append(values1, textData...)
			}
		}

		// Заполняем файл данными
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return fmt.Errorf("open %s file: %w", fileName, err)
		}
		_, err = file.WriteString(
			fmt.Sprintf(content,
				strings.Join(imports, "\n"),
				strings.Join(values1, "\n"),
				strings.Join(values2, "\n"),
			),
		)
		if err != nil {
			file.Close()
			return fmt.Errorf("write %s file: %w", fileName, err)
		}
		file.Close()
	}
	return nil
}
