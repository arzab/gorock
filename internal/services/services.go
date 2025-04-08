package services

import (
	_ "regexp"
	"github.com/arzab/gorock/internal/services/hello"
	//Imports
)

var (
	varHello hello.Service
	// Var
)

func Hello() hello.Service { return varHello }
	//Functions
