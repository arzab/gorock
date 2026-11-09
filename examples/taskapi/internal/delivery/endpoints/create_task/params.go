package create_task

import (
	"net/http"
	"strings"

	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"
)

type Params struct {
	Title string `json:"title"`
}

// Validate is called by rockfiber.DefaultHandler after parsing.
func (p *Params) Validate(*fiber.Ctx) error {
	if strings.TrimSpace(p.Title) == "" {
		return rockfiber.NewError(http.StatusBadRequest, "title is required")
	}
	return nil
}

// InitParams parses query, body, headers and path params into Params,
// validates them and stores them in ctx.Locals under "params".
func InitParams() fiber.Handler { return rockfiber.DefaultHandler[Params]() }

func GetParams(c *fiber.Ctx) (*Params, error) {
	return rockfiber.GetFromContext[Params](c, "params")
}
