package hello

import (
	"fmt"
	"github.com/arzab/gorock/pkg/http"
	"github.com/arzab/gorock/pkg/http/params"
	"github.com/gofiber/fiber/v2"
)

type Params struct {
}

func (params *Params) Validate(ctx *fiber.Ctx) error {
	return nil
}

func InitParams() fiber.Handler {
	return params.DefaultHandler[Params]()
}

func GetParams(ctx *fiber.Ctx) (*Params, error) {
	res, err := http.GetFromContext[Params](ctx, "params")
	if err != nil {
		return nil, fmt.Errorf("no params object in ctx locals, check handlers")
	}
	return res, nil
}
