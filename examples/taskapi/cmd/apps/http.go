package apps

import (
	"net/http"

	"github.com/arzab/gorock-catalog/rockhttp"

	"github.com/arzab/gorock/examples/taskapi/internal/handlers/tasks"
)

// httpEndpoints: which handler answers which method and path.
func httpEndpoints() []rockhttp.Endpoint {
	return []rockhttp.Endpoint{
		{Method: http.MethodPost, Path: "/tasks", Handler: tasks.Create, Status: http.StatusCreated},
		{Method: http.MethodGet, Path: "/tasks/{id}", Handler: tasks.Get},
	}
}
