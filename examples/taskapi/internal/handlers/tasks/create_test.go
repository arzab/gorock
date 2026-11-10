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

	res, err := tasks.Create.Call(context.Background(), &tasks.CreateParams{Title: " write docs "})
	if err != nil {
		t.Fatal(err)
	}
	task := res.(*tasks.CreateResponse).Task
	if task.Title != "write docs" || fake.tasks[task.ID] == nil {
		t.Fatalf("result %+v, stored %v", task, fake.tasks)
	}
}

func TestCreateEmptyTitle(t *testing.T) {
	useFake(t)

	_, err := tasks.Create.Call(context.Background(), &tasks.CreateParams{Title: "  "})
	if rockhandlers.KindOf(err) != rockhandlers.KindInvalid || rockhandlers.FieldsOf(err)["title"] != "required" {
		t.Fatalf("err = %v, fields %v", err, rockhandlers.FieldsOf(err))
	}
}

func TestCreateRepositoryFailure(t *testing.T) {
	fake := useFake(t)
	fake.err = errors.New("connection refused to 10.0.0.5:5432")

	_, err := tasks.Create.Call(context.Background(), &tasks.CreateParams{Title: "x"})
	if rockhandlers.KindOf(err) != rockhandlers.KindInternal {
		t.Fatalf("kind = %v", rockhandlers.KindOf(err))
	}
	if msg := rockhandlers.MessageOf(err); msg != "" {
		t.Fatalf("internal text reaches the client: %q", msg)
	}
}
