package server

import (
	"github.com/arzab/gorock/pkg/http/endpoints"
)

type Service interface {
	Init() error
	Exec(httpEndpoints []endpoints.FiberEndpoint) error
	Shutdown(shutdownFunc func() []error) []error
}
