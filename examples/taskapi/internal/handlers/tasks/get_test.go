package tasks_test

import (
	"context"
	"testing"

	"github.com/arzab/gorock-kit/rockhandlers"

	"github.com/arzab/gorock/examples/taskapi/internal/handlers/tasks"
	"github.com/arzab/gorock/examples/taskapi/internal/models"
)

func TestGet(t *testing.T) {
	fake := useFake(t)
	fake.tasks["42"] = &models.Task{ID: "42", Title: "write docs"}

	res, err := tasks.Get.Call(context.Background(), &tasks.GetParams{ID: "42"})
	if err != nil || res.(*tasks.GetResponse).Title != "write docs" {
		t.Fatalf("res %+v, err %v", res, err)
	}
}

func TestGetUnknown(t *testing.T) {
	useFake(t)

	_, err := tasks.Get.Call(context.Background(), &tasks.GetParams{ID: "nope"})
	if rockhandlers.KindOf(err) != rockhandlers.KindNotFound || rockhandlers.MessageOf(err) != "task not found" {
		t.Fatalf("err = %v", err)
	}
}
