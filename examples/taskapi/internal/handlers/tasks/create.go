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

// Validate runs before every handler: check first, use later.
func (p *CreateParams) Validate() error {
	if strings.TrimSpace(p.Title) == "" {
		return rockhandlers.Invalid("invalid task").WithFields(map[string]string{"title": "required"})
	}
	return nil
}

type CreateResponse struct {
	Task *models.Task `json:"task"`
}

func (r *CreateResponse) Normalize() error { return nil }

var Create = rockhandlers.Chain(create)

func create(c *rockhandlers.Ctx, params *CreateParams, response *CreateResponse) error {
	task := &models.Task{ID: uuid.NewString(), Title: strings.TrimSpace(params.Title), CreatedAt: time.Now().UTC()}
	if err := services.Repository().Insert(c, task); err != nil {
		return fmt.Errorf("insert task: %w", err) // unexpected: 500, message hidden
	}
	response.Task = task
	return nil
}
