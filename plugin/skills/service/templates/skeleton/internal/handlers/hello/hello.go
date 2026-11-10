package hello

import (
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
)

// GET /hello?name=Ann — a minimal handler. It knows nothing about HTTP: the
// App fills Params (query tag) and turns the Result into JSON.

type Params struct {
	Name string `query:"name"`
}

// Validate runs before every step: check first, use later.
func (p *Params) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return rockhandlers.Invalid("name is required")
	}
	return nil
}

type Result struct {
	Message string `json:"message"`
}

type Ctx = rockhandlers.Ctx[*Params, *Result]

var Hello = rockhandlers.Chain("hello", hello)

func hello(c *Ctx) error {
	c.Result.Message = "Hello, " + c.Params.Name
	return nil
}
