package configs

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func Init(configsPath string, configs interface{}) error {
	if reflect.TypeOf(configs).Kind() != reflect.Ptr {
		return fmt.Errorf("configs object must be ptr type")
	}

	ext := filepath.Ext(configsPath)

	if ext != ".json" {
		return fmt.Errorf("configs file must have .json extension")
	}

	data, err := os.ReadFile(configsPath)
	if err != nil {
		return fmt.Errorf("read data err: %v", err)
	}

	err = unmarshall(data, configs, ext)
	if err != nil {
		return fmt.Errorf("failed to unmarshall: %v", err)
	}

	emptyProperties := getEmptyProperties(configs, "Config")
	if len(emptyProperties) > 0 {
		for _, property := range emptyProperties {
			fmt.Printf("empty property: %s\n", property)
		}
		return fmt.Errorf("config struct have empty properties")
	}

	return nil
}

func unmarshall(data []byte, obj interface{}, ext string) error {
	dataStr := os.ExpandEnv(string(data))
	dataMap := make(map[string]interface{})

	data, err := io.ReadAll(strings.NewReader(dataStr))
	if err != nil {
		return fmt.Errorf("failed to read data from expanded string: %v", err)
	}

	if ext := filepath.Ext(ext); ext == ".json" {
		err = json.Unmarshal(data, &dataMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshall json file data into map: %v", err)
		}

		data, err = json.Marshal(dataMap)
		if err != nil {
			return fmt.Errorf("failed to read json file: %v", err)
		}

		err = json.Unmarshal(data, obj)
		if err != nil {
			return fmt.Errorf("failed to marshall json file into configs object: %v", err)
		}
	} else if ext == ".yaml" {
		err = yaml.Unmarshal(data, &dataMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshall yaml file data into map: %v", err)
		}

		data, err = yaml.Marshal(dataMap)
		if err != nil {
			return fmt.Errorf("failed to read yaml file: %v", err)
		}

		err = yaml.Unmarshal(data, obj)
		if err != nil {
			return fmt.Errorf("failed to marshall yaml file into configs object: %v", err)
		}
	} else {
		return fmt.Errorf("unknown extension: %s", ext)
	}
	return nil
}

func getEmptyProperties(object interface{}, structName string) []string {
	properties := make([]string, 0)
	value := reflect.ValueOf(object)

	value, isStruct := getStructValue(value)
	if !isStruct {
		return nil
	}

	if value.NumField() == 0 {
		properties = append(properties, structName)
		return properties
	}

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldName := value.Type().Field(i).Name
		tag := value.Type().Field(i).Tag

		if key := tag.Get("config"); key == "ignore" {
			continue
		}
		if key := tag.Get("config"); key == "omit_empty" {
			if field.IsNil() {
				continue
			}
		}

		field, isStruct = getStructValue(field)
		if isStruct {
			if field.NumField() == 0 {
				continue
			}

			indirectProperties := getEmptyProperties(field.Interface(), fieldName)
			for _, property := range indirectProperties {
				properties = append(properties, fmt.Sprintf("%s -> %s", structName, property))
			}
			continue
		}

		if valueIsEmpty(value.Field(i)) {
			properties = append(properties, fmt.Sprintf("%s -> %s", structName, fieldName))
		}
	}
	return properties
}

func getStructValue(value reflect.Value) (reflect.Value, bool) {
	if value.Kind() == reflect.Ptr {
		value = reflect.Indirect(value)
	}

	if value.Kind() != reflect.Struct {
		return reflect.ValueOf(nil), false
	}

	return value, true
}

func valueIsEmpty(value reflect.Value) bool {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		return true
	} else if value.Kind() == reflect.Ptr {
		value = reflect.Indirect(value)
	}

	switch value.Kind() {
	case reflect.String:
		return value.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Int() == 0
	case reflect.Float32, reflect.Float64:
		return value.Int() == 0
	case reflect.Array, reflect.Map, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
