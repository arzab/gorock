package endpoints

import (
	"net/http"

	"github.com/arzab/gorock-catalog/rockhttp"

	"__MODULE__/internal/delivery/endpoints/hello"
)

// All lists every HTTP endpoint: where each rockhandlers route is mounted.
// Method, path and success status are HTTP things, so they live here, not in
// the route.
func All() []rockhttp.Endpoint {
	return []rockhttp.Endpoint{
		{Method: http.MethodGet, Path: "/hello", Handler: hello.Route},
	}
}
