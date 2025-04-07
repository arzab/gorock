package http

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
)

func GetFromContext[T any](ctx *fiber.Ctx, keyStr string) (*T, error) {
	obj := ctx.Locals(keyStr)
	if obj == nil {
		return nil, fmt.Errorf("%s not found in ctx locals", keyStr)
	}

	result, ok := obj.(*T)
	if !ok {
		return nil, fmt.Errorf("invalid type of params object")
	}

	return result, nil
}

func HandlerInitInCtx[T any](keyStr string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		obj := new(T)
		ctx.Locals(keyStr, obj)
		return ctx.Next()
	}
}
