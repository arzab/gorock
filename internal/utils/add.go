package utils

import (
	"bytes"
	"fmt"
	"github.com/spf13/afero"
	"os"
	"regexp"
)

var regexImport = regexp.MustCompile(`import \((\n|[^)])+`)

type Operation int
type OperationFunc func(content []byte, find string, data string) []byte

const (
	AddAfter Operation = iota
	AddBefore
	Replace
)

type OperationArgs struct {
	Find  string
	Value string
}

type DataUpdater map[Operation][]OperationArgs

func (d DataUpdater) Update(data []byte) []byte {
	for key, args := range d {
		switch key {
		case AddAfter:
			for _, arg := range args {
				data = AddAfterValue(data, arg)
			}
		case AddBefore:
			for _, arg := range args {
				data = AddBeforeValue(data, arg)
			}
		case Replace:
			for _, arg := range args {
				data = ReplaceValue(data, arg)
			}
		}
	}

	return data
}

func AddAfterValue(content []byte, args OperationArgs) []byte {
	content = bytes.ReplaceAll(content, []byte(args.Find), []byte(fmt.Sprintf("%s\n\t%s", args.Find, args.Value)))
	return content
}
func AddBeforeValue(content []byte, args OperationArgs) []byte {
	content = bytes.ReplaceAll(content, []byte(args.Find), []byte(fmt.Sprintf("%s\n\t%s", args.Value, args.Find)))
	return content
}
func ReplaceValue(content []byte, args OperationArgs) []byte {
	content = bytes.ReplaceAll(content, []byte(args.Find), []byte(args.Value))
	return content
}

func CheckDir(fs afero.Fs, path string, mustExists, force bool) error {
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

func CreateFromDir(fs afero.Fs, src, dest string, dataUpdater DataUpdater) error {
	err := CheckDir(fs, dest, false, false)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s dir already exists", dest)
		}
		return err
	}
	err = fs.Mkdir(dest, 0755)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", dest, err)
	}
	// get template endpoint
	dirEntry, err := afero.ReadDir(fs, src)
	if err != nil {
		return fmt.Errorf("read example dir: %w", err)
	}
	// copy files from templates to endpoint
	for _, info := range dirEntry {
		if info.IsDir() {
			continue
		}
		err = CreateFromFile(
			fs,
			fmt.Sprintf("%s/%s", src, info.Name()),
			fmt.Sprintf("%s/%s", dest, info.Name()),
			dataUpdater,
		)
		if err != nil {
			return fmt.Errorf("create %s: %w", info.Name(), err)
		}
	}
	return nil
}

func CreateFromFile(fs afero.Fs, src, dest string, dataUpdater DataUpdater) error {
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

	data = dataUpdater.Update(data)

	err = afero.WriteFile(fs, dest, data, 0755)
	if err != nil {
		return fmt.Errorf("copy %s to %s: %w", src, dest, err)
	}
	return nil
}
