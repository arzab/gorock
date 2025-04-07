package params

import "github.com/gofiber/fiber/v2"

type Service[T any] interface {
	*T
	Validate(ctx *fiber.Ctx) error
}
