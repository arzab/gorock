package server

import (
	"fmt"
	"github.com/arzab/gorock/pkg/http/endpoints"
	"github.com/arzab/gorock/pkg/http/server/services/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type service struct {
	configs Configs
	handler handler.Service
	app     *fiber.App
}

func (s *service) Init() error {
	if s.configs.AdminEndpointsPath == "" {
		s.configs.AdminEndpointsPath = "/admin"
	} else {
		s.configs.AdminEndpointsPath = strings.TrimRight(s.configs.AdminEndpointsPath, "/")
		s.configs.AdminEndpointsPath = strings.TrimLeft(s.configs.AdminEndpointsPath, "/")
		s.configs.AdminEndpointsPath = fmt.Sprintf("/%s", s.configs.AdminEndpointsPath)
	}
	appConfigs := s.configs.App
	appConfigs.ErrorHandler = s.handler.ErrorHandler()

	s.app = fiber.New(appConfigs)

	// Настройка recover обработчика при получении panic
	s.app.Use(s.handler.RecoverHandler())

	// Настройка обработчика для профилирования
	s.app.Use(s.handler.PprofHandler(s.configs.AdminEndpointsPath))

	// Настройка CORS
	s.app.Use(s.handler.CorsHandler())

	return nil
}

func (s *service) Exec(httpEndpoints []endpoints.FiberEndpoint) error {
	s.setupHttpEndpoints(s.app, httpEndpoints)

	return s.app.Listen(fmt.Sprintf(":%s", s.configs.Port))
}

func (s *service) Shutdown(shutdownFunc func() []error) []error {
	errors := make([]error, 0)

	err := s.app.ShutdownWithTimeout(time.Duration(time.Second * 30))
	if err != nil {
		errors = append(errors, fmt.Errorf("failed shutdown app: %v", err))
	}

	errors = append(errors, shutdownFunc()...)

	return errors
}

func (s *service) setupHttpEndpoints(app fiber.Router, endpoints []endpoints.FiberEndpoint) {
	// Setting up swagger configs
	if s.configs.Swagger != nil {
		app.Get(s.configs.Swagger.Path, swagger.HandlerDefault)
		swaggerConfigs := s.configs.Swagger.Configs
		swaggerConfigs.OAuth = s.configs.Swagger.OAuth
		swaggerConfigs.Filter = s.configs.Swagger.Filter
		swaggerConfigs.SyntaxHighlight = s.configs.Swagger.SyntaxHighlight
		app.Get(fmt.Sprintf("%s/*", s.configs.Swagger.Path), swagger.New(swaggerConfigs))
	}

	// Settings up admin endpoints
	admin := app.Group(s.configs.AdminEndpointsPath)
	admin.Use(s.handler.AdminAuthMiddleware(s.configs.AdminPassword))
	admin.Get("/metrics", s.handler.MetricsHandler())

	// Set alive check endpoint
	app.Get("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(map[string]interface{}{
			"status": "ok",
		})
	})

	// Настраваем websocket эндпоинты
	//webSocketRouter := app.Group(s.configs.WebsocketPath)
	//webSocketRouter.Use("", handlerService.WebsocketMiddleware())
	//for _, endpoint := range websocketEndpoints {
	//	SetupEndpoint(webSocketRouter, endpoint)
	//}

	// Set http endpoints
	var httpEndpointsRouter fiber.Router

	if len(s.configs.EndpointsPathPrefix) > 0 {
		httpEndpointsRouter = app.Group(s.configs.EndpointsPathPrefix)
	} else {
		httpEndpointsRouter = app.Group("/")
	}
	if s.configs.UseTraceId {
		httpEndpointsRouter.Use(s.handler.GenerateTraceIdMiddleware())
	}
	if s.configs.LogRequestReceive {
		httpEndpointsRouter.Use(s.handler.RequestReceiverMiddleware())
	}
	for _, endpoint := range endpoints {
		setupEndpoint(httpEndpointsRouter, endpoint)
	}

	// Print registered endpoints
	fmt.Println("\n\nRoutes:")
	var endpointPathReg = regexp.MustCompile(fmt.Sprintf("^(%s){1}/.+", s.configs.EndpointsPathPrefix))
	for _, route := range s.app.GetRoutes(true) {
		if endpointPathReg.MatchString(route.Path) {
			fmt.Println(route.Method, " : ", route.Path)
		}
	}
	return
}

func NewService(configs Configs) Service {
	return &service{
		configs: configs,
		handler: handler.NewService(configs.Handler),
	}
}

func setupEndpoint(router fiber.Router, endpoint endpoints.Service[fiber.Handler]) {
	switch strings.ToUpper(endpoint.GetMethod()) {
	case http.MethodGet:
		router.Get(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodHead:
		router.Head(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodPost:
		router.Post(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodPut:
		router.Put(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodPatch:
		router.Patch(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodDelete:
		router.Delete(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodConnect:
		router.Connect(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodOptions:
		router.Options(endpoint.GetPath(), endpoint.GetHandlers()...)
	case http.MethodTrace:
		router.Trace(endpoint.GetPath(), endpoint.GetHandlers()...)
	default:
		router.All(endpoint.GetPath(), endpoint.GetHandlers()...)
	}
}
