package tasks_test

import (
	"context"
	"testing"

	"github.com/arzab/gorock-kit/rockservices"

	"github.com/arzab/gorock/examples/taskapi/internal/models"
	"github.com/arzab/gorock/examples/taskapi/pkg/services"
	"github.com/arzab/gorock/examples/taskapi/pkg/services/repository"
)

// fakeRepository stores tasks in memory; err, when set, is returned by every call.
type fakeRepository struct {
	tasks map[string]*models.Task
	err   error
}

func (f *fakeRepository) Init(context.Context) error { return nil }
func (f *fakeRepository) Stop() error                { return nil }

func (f *fakeRepository) Insert(_ context.Context, model any) error {
	if f.err != nil {
		return f.err
	}
	t := model.(*models.Task)
	f.tasks[t.ID] = t
	return nil
}

func (f *fakeRepository) GetByID(_ context.Context, model any, id string) error {
	if f.err != nil {
		return f.err
	}
	t, ok := f.tasks[id]
	if !ok {
		return repository.ErrNotFound
	}
	*model.(*models.Task) = *t
	return nil
}

// useFake makes a catalog with the fake the default one for this test.
// Tests that swap the default catalog must not use t.Parallel().
func useFake(t *testing.T) *fakeRepository {
	t.Helper()
	fake := &fakeRepository{tasks: map[string]*models.Task{}}
	c := rockservices.New()
	rockservices.Add(c, services.Repository, func(struct{}) repository.Service { return fake }, struct{}{})
	prev := rockservices.SetDefault(c)
	t.Cleanup(func() { rockservices.SetDefault(prev) })
	return fake
}
