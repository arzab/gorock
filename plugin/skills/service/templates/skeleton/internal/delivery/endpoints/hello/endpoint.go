package hello

import (
	"net/http"
	"strings"

	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"
)

// GET /hello?name=Ann — a minimal self-describing endpoint.

type Params struct {
	Name string `query:"name"`
}

func (p *Params) Validate(*fiber.Ctx) error {
	if strings.TrimSpace(p.Name) == "" {
		return rockfiber.NewError(http.StatusBadRequest, "name is required")
	}
	return nil
}

type Response struct {
	Message string `json:"message"`
}

func Endpoint() rockfiber.FiberEndpoint {
	return rockfiber.GET("/hello", rockfiber.DefaultHandler[Params](), hello())
}

func hello() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params, err := rockfiber.GetFromContext[Params](c, "params")
		if err != nil {
			return err
		}
		return c.JSON(Response{Message: "Hello, " + params.Name})
	}
}
