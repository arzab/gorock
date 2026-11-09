package get_task

import (
	"errors"
	"net/http"

	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/gofiber/fiber/v2"

	"github.com/yourname/taskapi/internal/models"
	"github.com/yourname/taskapi/pkg/services"
	"github.com/yourname/taskapi/pkg/services/repository"
)

// A small endpoint can live in one file.

type Params struct {
	ID string `params:"id"`
}

func (p *Params) Validate(*fiber.Ctx) error {
	if p.ID == "" {
		return rockfiber.NewError(http.StatusBadRequest, "id is required")
	}
	return nil
}

func Endpoint() rockfiber.FiberEndpoint {
	return rockfiber.GET("/tasks/:id", rockfiber.DefaultHandler[Params](), getTask())
}

func getTask() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params, err := rockfiber.GetFromContext[Params](c, "params")
		if err != nil {
			return err
		}

		task := &models.Task{}
		err = services.Repository().GetByID(c.UserContext(), task, params.ID)
		if errors.Is(err, repository.ErrNotFound) {
			return rockfiber.NewError(http.StatusNotFound, "task not found") // business decision: 404
		}
		if err != nil {
			return err // unexpected: 500
		}
		return c.JSON(task)
	}
}
