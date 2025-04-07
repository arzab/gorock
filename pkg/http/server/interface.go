package server

import (
	"github.com/arzab/gorock/pkg/http/endpoints"
	"github.com/gofiber/fiber/v2"
)

type Service interface {
	Init() error
	Exec(httpEndpoints []endpoints.Service[fiber.Handler]) error
	Shutdown(shutdownFunc func() []error) []error
}
