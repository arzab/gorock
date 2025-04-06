package handler

import "github.com/gofiber/fiber/v2"

type Service interface {
	ErrorHandler() func(ctx *fiber.Ctx, err error) error
	RecoverHandler() fiber.Handler
	PprofHandler(prefix string) fiber.Handler
	MetricsHandler() fiber.Handler
	CorsHandler() fiber.Handler
	EmptyRouteMiddleware() fiber.Handler
	GenerateTraceIdMiddleware() fiber.Handler
	AdminAuthMiddleware(password string) fiber.Handler
	RequestReceiverMiddleware() fiber.Handler
	//WebsocketMiddleware() fiber.Handler
}
