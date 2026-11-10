package hello

import (
	"strings"

	"github.com/arzab/gorock-kit/rockhandlers"
)

// GET /hello?name=Ann — a minimal chain. It knows nothing about HTTP: the
// App fills Params (query tag) and turns the Response into JSON.

type Params struct {
	Name string `query:"name"`
}

// Validate runs before every handler: check first, use later.
func (p *Params) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return rockhandlers.Invalid("name is required")
	}
	return nil
}

type Response struct {
	Message string `json:"message"`
}

// Normalize runs after the chain, if there was no error.
func (r *Response) Normalize() error { return nil }

var Hello = rockhandlers.Chain(hello)

func hello(_ *rockhandlers.Ctx, params *Params, response *Response) error {
	response.Message = "Hello, " + params.Name
	return nil // done; return c.Next() would run the next handler
}
