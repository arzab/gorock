package events

import (
	"context"

	"github.com/arzab/gorock-kit/rockbus"
	"github.com/arzab/gorock-kit/rocklog"

	"github.com/yourname/taskapi/internal/models"
)

const TaskCreated rockbus.Topic = "task.created"

// Subscriptions lists every event handler. The bus App registers them.
func Subscriptions() []rockbus.Subscription {
	return []rockbus.Subscription{
		rockbus.On(TaskCreated, onTaskCreated),
	}
}

func onTaskCreated(ctx context.Context) error {
	task, err := rockbus.Payload[models.Task](ctx)
	if err != nil {
		return err
	}
	rocklog.FromContext(ctx).Info("task created", rocklog.Str("id", task.ID), rocklog.Str("title", task.Title))
	return nil
}
