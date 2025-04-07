package endpoints

import "github.com/gofiber/fiber/v2"

type Service[HandlerFunc any] interface {
	GetPath() string
	GetMethod() string
	GetHandlers() []HandlerFunc
}

type FiberEndpoint Service[fiber.Handler]
