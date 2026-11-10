package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/arzab/gorock-kit/rockhandlers"
	"github.com/google/uuid"

	"github.com/arzab/gorock/examples/taskapi/internal/models"
	"github.com/arzab/gorock/examples/taskapi/pkg/services"
)

type CreateParams struct {
	Title string `json:"title"`
}

// Validate runs before every step: check first, use later.
func (p *CreateParams) Validate() error {
	if strings.TrimSpace(p.Title) == "" {
		return rockhandlers.Invalid("invalid task").WithFields(map[string]string{"title": "required"})
	}
	return nil
}

type CreateResult struct {
	Task *models.Task `json:"task"`
}

var Create = rockhandlers.Chain("tasks.create", create)

func create(c *rockhandlers.Ctx[*CreateParams, *CreateResult]) error {
	task := &models.Task{ID: uuid.NewString(), Title: strings.TrimSpace(c.Params.Title), CreatedAt: time.Now().UTC()}
	if err := services.Repository.Get().Insert(c, task); err != nil {
		return fmt.Errorf("insert task: %w", err) // unexpected: 500, message hidden
	}
	c.Result.Task = task
	return nil
}
