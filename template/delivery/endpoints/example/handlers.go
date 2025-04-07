package example

import (
	"github.com/arzab/gorock/pkg/http/params"
	"github.com/gofiber/fiber/v2"
)

func handlers() []fiber.Handler {
	return []fiber.Handler{
		params.DefaultHandler[Params](),
		InitResponse(),
		// Handlers
		// Handlers
		returnResponse(),
	}
}

func returnResponse() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		resp, err := GetResponse(ctx)
		if err != nil {
			return err
		}
		return ctx.JSON(resp)
	}
}
