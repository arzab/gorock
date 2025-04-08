package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/arzab/gorock/internal/commands"
	"github.com/arzab/gorock/internal/utils"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"path/filepath"
)

func main() {

	// Копируем шаблоны в нужную директорию
	err := initTemplateDir()
	if err != nil {
		log.Fatal(err)
	}

	cmd := &cli.Command{
		Name:      "gorock",
		Aliases:   []string{"gorock"},
		Commands:  commands.Commands,
		ErrWriter: os.Stdout,
	}

	err = cmd.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
	}
}

//go:embed template/**
var embeddedTemplates embed.FS

func initTemplateDir() error {
	// Находим домашнюю директорию
	homeDirPath, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %s", err)
	}

	// Путь куда будут копироваться шаблоны
	templateDirPath := filepath.Join(homeDirPath, ".gorock", "template")
	utils.TemplateDirPath = templateDirPath

	// Проверяем, существует ли директория ~/.gorock/template
	//if _, err := os.Stat(templateDirPath); err == nil {
	//	// Директория уже существует, пропускаем копирование
	//	return nil
	//}

	// Создаём директорию ~/.gorock/template
	err = os.MkdirAll(templateDirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("could not create template directory: %s", err)
	}

	// Проходим по всем файлам и папкам в embeddedTemplates
	err = copyEmbedDir("template", templateDirPath)
	if err != nil {
		return fmt.Errorf("could not copy embedded template directory: %s", err)
	}

	return nil
}

func copyEmbedDir(srcDir, destDir string) error {
	// Читаем содержимое директории
	entries, err := embeddedTemplates.ReadDir(srcDir)
	if err != nil {
		fmt.Println("could not read embedded template directory:", err)
		return fmt.Errorf("could not read embedded directory: %s", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(destDir, entry.Name())

		if entry.IsDir() {
			// Создаём папку
			err := os.MkdirAll(destPath, 0755)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("could not create directory %s: %s", destPath, err)
			}
			// Рекурсивно копируем содержимое директории
			if err := copyEmbedDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if _, err := os.Stat(destPath); !os.IsNotExist(err) {
				continue
			}

			// Копируем файл через ReadFile и WriteFile
			data, err := embeddedTemplates.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("could not read embedded file %s: %s", srcPath, err)
			}

			err = os.WriteFile(destPath, data, 0755)
			if err != nil {
				return fmt.Errorf("could not write file %s: %s", destPath, err)
			}
		}
	}

	return nil
}
