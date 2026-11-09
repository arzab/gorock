package hello

import (
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
)

// GET /hello?name=Ann — a minimal route. It knows nothing about HTTP: the
// transport fills Params (query tag) and turns the Result into JSON.

type Params struct {
	Name string `query:"name"`
}

// Validate runs right after the request is decoded, before the steps.
func (p *Params) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return rockhandlers.Invalid("name is required") // → 400
	}
	return nil
}

type Result struct {
	Message string `json:"message"`
}

type Ctx = rockhandlers.Ctx[*Params, *Result]

var Route = rockhandlers.Chain("hello", hello)

func hello(c *Ctx) error {
	c.Result.Message = "Hello, " + c.Params.Name
	return nil
}
