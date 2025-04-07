package endpoints

import "github.com/gofiber/fiber/v2"

type fiberEndpoint struct {
	Method   string
	Path     string
	Handlers []fiber.Handler
}

func (f fiberEndpoint) GetPath() string {
	return f.Path
}

func (f fiberEndpoint) GetMethod() string {
	return f.Method
}

func (f fiberEndpoint) GetHandlers() []fiber.Handler {
	return f.Handlers
}

func BuildFiberEndpoint(method, path string, handlers []fiber.Handler) FiberEndpoint {
	return &fiberEndpoint{
		Method:   method,
		Path:     path,
		Handlers: handlers,
	}
}
