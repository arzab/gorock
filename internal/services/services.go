package services

import (
	"github.com/arzab/gorock/internal/services/asd"
	"github.com/arzab/gorock/internal/services/asfaf"
	"github.com/arzab/gorock/internal/services/asfasfd"
	_ "regexp"
	//Imports
)

var (
	varAsd     asd.Service
	varAsfasfd asfasfd.Service
	varAsfaf   asfaf.Service
	// Var
)

func Asd() asd.Service         { return varAsd }
func Asfasfd() asfasfd.Service { return varAsfasfd }
func Asfaf() asfaf.Service     { return varAsfaf }

//Functions
