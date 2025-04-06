package handler

import (
	"errors"
	"fmt"
	"github.com/arzab/gorock/pkg/http/responses"
	"github.com/arzab/gorock/pkg/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"net/http"
	"os"
	"runtime/debug"
)

type service struct {
	configs Configs
}

func NewService(configs Configs) Service {
	return &service{
		configs: configs,
	}
}

const TraceIdKey = "X-Trace-Id"

func (s service) ErrorHandler() func(ctx *fiber.Ctx, err error) error {
	return func(ctx *fiber.Ctx, err error) error {
		var errorResponse *responses.ErrorResponse
		var fiberError *fiber.Error

		// Get debug query param to ignore masking error if we need
		debugValue := ctx.Query("debug")
		// If in configs we have not masked message, we show error in response
		showError := len(debugValue) == 0 || len(s.configs.MaskInternalServerErrorMessage) == 0

		if errors.As(err, &errorResponse) { // Check if handlerFunc returned gorock typed error
			// nothing to do, cause in end of function we return errorResponse
			// But we must handle it case errors.As(err, &fiberError) working with errorResponse
		} else if errors.As(err, &fiberError) { // Check if handlerFunc returned fiber typed error
			errorResponse = responses.NewError(fiberError.Code, fiberError.Message)
		} else {
			if showError {
				errorResponse = responses.NewError(http.StatusInternalServerError, err.Error(), "internal", "unknown")
			} else {
				errorResponse = responses.NewError(http.StatusInternalServerError, s.configs.MaskInternalServerErrorMessage, "internal", "unknown")
			}
		}

		// Check do we need to log the error
		if !s.configs.IgnoreLogError {
			logMessage := map[string]interface{}{}

			// Getting Trace id to show in log
			traceIdValue := ctx.Locals(TraceIdKey)
			if traceIdValue == nil {
				traceIdValue = ""
			}
			traceId := traceIdValue.(string)

			if len(traceId) > 0 {
				logMessage["trace_id"] = traceId
			}

			// hope errorResponse will not be nil, cause we filled it
			logMessage["error"] = errorResponse

			log.ErrorWithFields(logMessage, "request failed")
		}

		return ctx.Status(errorResponse.Code).JSON(errorResponse)
	}
}
func (s service) RecoverHandler() fiber.Handler {
	return recover.New(recover.Config{
		Next:             nil,
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", e, debug.Stack())) //nolint:errcheck // This will never fail
		},
	})
}

func (s service) PprofHandler(prefix string) fiber.Handler {
	return pprof.New(pprof.Config{
		Next:   nil,
		Prefix: prefix,
	})
}

func (s service) CorsHandler() fiber.Handler {
	if s.configs.CorsConfigs != nil {
		return cors.New(*s.configs.CorsConfigs)
	}
	return cors.New()
}

var defaultMonitoringConfig = monitor.Config{
	Title:      "Monitoring",
	Refresh:    1,
	APIOnly:    false,
	Next:       nil,
	CustomHead: "",
	FontURL:    "",
	ChartJsURL: "",
}

func (s service) MetricsHandler() fiber.Handler {
	if s.configs.MonitoringConfigs != nil {
		return monitor.New(*s.configs.MonitoringConfigs)
	} else {
		return monitor.New(defaultMonitoringConfig)
	}
}

func (s service) EmptyRouteMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if ctx.Route() != nil {
			return ctx.Next()
		}

		return responses.NewError(http.StatusNotImplemented, "endpoint registered, but handler returned nil")
	}
}

func (s service) GenerateTraceIdMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		headers := ctx.GetReqHeaders()
		if headers == nil {
			return ctx.Next()
		}
		if _, ok := headers["Trace-Id"]; !ok {
			ctx.GetReqHeaders()["Trace-Id"] = []string{uuid.NewString()}
		}

		return ctx.Next()
	}
}

func (s service) AdminAuthMiddleware(password string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		query := ctx.Query("password")
		if query != password {
			return responses.NewError(http.StatusUnauthorized, "invalid password")
		}
		return ctx.Next()
	}
}

func (s service) RequestReceiverMiddleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		path := ctx.Request().RequestURI()
		traceId := ctx.Locals(TraceIdKey)

		logMessage := map[string]interface{}{}

		if traceId != nil {
			logMessage["trace_id"] = traceId
		}
		logMessage["path"] = path

		log.InfoWithFields(logMessage, "request received")
		return ctx.Next()
	}
}

//
//func (s service) WebsocketMiddleware() fiber.Handler {
//	return func(ctx *fiber.Ctx) error {
//		if websocket.IsWebSocketUpgrade(ctx) {
//			ctx.Locals("allowed", true)
//			return ctx.Next()
//		}
//		return fiber.ErrUpgradeRequired
//	}
//
//}
