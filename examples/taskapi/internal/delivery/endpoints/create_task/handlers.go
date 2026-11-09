package create_task

import (
	"fmt"
	"net/http"

	"github.com/arzab/gorock-kit/rockbus"
	"github.com/arzab/gorock-kit/rockfiber"
	"github.com/arzab/gorock-kit/rocklog"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/yourname/taskapi/internal/delivery/events"
	"github.com/yourname/taskapi/internal/models"
	"github.com/yourname/taskapi/pkg/services"
)

// handlers is the chain: each function does one step and calls c.Next().
func handlers() []fiber.Handler {
	return []fiber.Handler{
		InitParams(),   // parse + validate input
		InitResponse(), // prepare the response object
		createTask(),   // business logic
		sendResponse(), // serialise
	}
}

func createTask() fiber.Handler {
	return func(c *fiber.Ctx) error {
		params, err := GetParams(c)
		if err != nil {
			return err
		}
		resp, err := GetResponse(c)
		if err != nil {
			return err
		}

		// Request-scoped logger carrying the trace id; it travels with ctx
		// into the event handler as well.
		ctx := rocklog.WithContext(c.UserContext(),
			rocklog.Str("trace_id", rockfiber.TraceIdFromContext(c.UserContext())))

		task := &models.Task{ID: uuid.NewString(), Title: params.Title}
		if err := services.Repository().Insert(ctx, task); err != nil {
			// A plain error becomes a 500; its text is masked for the client.
			return fmt.Errorf("insert task: %w", err)
		}

		rockbus.PublishAsync(ctx, rockbus.Event{Topic: events.TaskCreated, Payload: *task})
		resp.Task = task
		return c.Next()
	}
}

func sendResponse() fiber.Handler {
	return func(c *fiber.Ctx) error {
		resp, err := GetResponse(c)
		if err != nil {
			return err
		}
		return c.Status(http.StatusCreated).JSON(resp)
	}
}
