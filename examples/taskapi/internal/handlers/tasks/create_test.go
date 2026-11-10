package tasks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arzab/gorock-kit/rockhandlers"

	"github.com/arzab/gorock/examples/taskapi/internal/handlers/tasks"
)

func TestCreate(t *testing.T) {
	fake := useFake(t)

	res, err := rockhandlers.Run[*tasks.CreateResult](context.Background(), tasks.Create, &tasks.CreateParams{Title: " write docs "})
	if err != nil {
		t.Fatal(err)
	}
	if res.Task.Title != "write docs" || fake.tasks[res.Task.ID] == nil {
		t.Fatalf("result %+v, stored %v", res.Task, fake.tasks)
	}
}

func TestCreateEmptyTitle(t *testing.T) {
	useFake(t)

	_, err := rockhandlers.Run[*tasks.CreateResult](context.Background(), tasks.Create, &tasks.CreateParams{Title: "  "})
	if rockhandlers.KindOf(err) != rockhandlers.KindInvalid || rockhandlers.FieldsOf(err)["title"] != "required" {
		t.Fatalf("err = %v, fields %v", err, rockhandlers.FieldsOf(err))
	}
}

func TestCreateRepositoryFailure(t *testing.T) {
	fake := useFake(t)
	fake.err = errors.New("connection refused to 10.0.0.5:5432")

	_, err := rockhandlers.Run[*tasks.CreateResult](context.Background(), tasks.Create, &tasks.CreateParams{Title: "x"})
	if rockhandlers.KindOf(err) != rockhandlers.KindInternal {
		t.Fatalf("kind = %v", rockhandlers.KindOf(err))
	}
	if msg := rockhandlers.MessageOf(err); msg != "" {
		t.Fatalf("internal text reaches the client: %q", msg)
	}
}
