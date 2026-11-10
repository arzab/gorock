package apps

import (
	"net/http"

	"github.com/arzab/gorock-catalog/rockhttp"

	"__MODULE__/internal/handlers/hello"
)

// httpEndpoints: which handler answers which method and path. Method, path
// and success status are HTTP things, so they live here, not in the handler.
func httpEndpoints() []rockhttp.Endpoint {
	return []rockhttp.Endpoint{
		{Method: http.MethodGet, Path: "/hello", Handler: hello.Hello},
	}
}
