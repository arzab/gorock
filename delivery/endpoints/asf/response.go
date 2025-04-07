package asf

import (
	"fmt"
	"github.com/arzab/gorock/pkg/http"
	"github.com/gofiber/fiber/v2"
)

type Response struct {
}

func InitResponse() fiber.Handler {
	return http.HandlerInitInCtx[Response]("response")
}

func GetResponse(ctx *fiber.Ctx) (*Response, error) {
	res, err := http.GetFromContext[Response](ctx, "response")
	if err != nil {
		return nil, fmt.Errorf("no response object in ctx locals, check handlers")
	}
	return res, nil
}
