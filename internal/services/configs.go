package services

import (
	_ "regexp"
	"github.com/arzab/gorock/internal/services/hello"
	//Imports
)

type Configs struct {
	Hello hello.Configs `json:"hello"`
	//Configs
}
