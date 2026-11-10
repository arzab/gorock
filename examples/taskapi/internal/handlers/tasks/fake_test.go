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

// useFake swaps the services catalog for one with the fake repository.
// Tests that swap the catalog must not use t.Parallel().
func useFake(t *testing.T) *fakeRepository {
	t.Helper()
	fake := &fakeRepository{tasks: map[string]*models.Task{}}
	t.Cleanup(services.UseCatalog(rockservices.New(map[string]rockservices.Service{
		services.RepositoryName: fake,
	})))
	return fake
}
