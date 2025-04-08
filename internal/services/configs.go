package services

import (
	_ "regexp"
	"github.com/arzab/gorock/internal/services/asd"
	"github.com/arzab/gorock/internal/services/asfasfd"
	"github.com/arzab/gorock/internal/services/asfaf"
	//Imports
)

type Configs struct {
	Asd asd.Configs `json:"asd"`
	Asfasfd asfasfd.Configs `json:"asfasfd"`
	Asfaf asfaf.Configs `json:"asfaf"`
	//Configs
}
