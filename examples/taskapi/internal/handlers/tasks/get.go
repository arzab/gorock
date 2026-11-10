package tasks

import (
	"errors"

	"github.com/arzab/gorock-kit/rockhandlers"

	"github.com/arzab/gorock/examples/taskapi/internal/models"
	"github.com/arzab/gorock/examples/taskapi/pkg/services"
	"github.com/arzab/gorock/examples/taskapi/pkg/services/repository"
)

type GetParams struct {
	ID string `path:"id"`
}

var Get = rockhandlers.Chain("tasks.get", get)

func get(c *rockhandlers.Ctx[*GetParams, *models.Task]) error {
	err := services.Repository.Get().GetByID(c, c.Result, c.Params.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return rockhandlers.NotFound("task not found")
	}
	return err
}
