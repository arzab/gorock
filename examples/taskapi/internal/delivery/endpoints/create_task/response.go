package create_task

import (
	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"

	"github.com/yourname/taskapi/internal/models"
)

type Response struct {
	Task *models.Task `json:"task"`
}

const responseKey = "response"

func InitResponse() fiber.Handler { return rockfiber.HandlerInitInCtx[Response](responseKey) }

func GetResponse(c *fiber.Ctx) (*Response, error) {
	return rockfiber.GetFromContext[Response](c, responseKey)
}
