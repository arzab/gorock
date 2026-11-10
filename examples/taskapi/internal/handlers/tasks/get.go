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

func (p *GetParams) Validate() error { return nil }

// GetResponse is the task itself: its fields are the JSON body.
type GetResponse struct {
	models.Task
}

func (r *GetResponse) Normalize() error { return nil }

var Get = rockhandlers.Chain(get)

func get(c *rockhandlers.Ctx, params *GetParams, response *GetResponse) error {
	err := services.Repository().GetByID(c, &response.Task, params.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return rockhandlers.NotFound("task not found")
	}
	return err
}
