package app

import (
	"github.com/arzab/gorock/pkg/http/app/services/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

type SwaggerConfigs struct {
	Path            string                         `json:"path"`
	Configs         swagger.Config                 `json:"configs"`
	OAuth           *swagger.OAuthConfig           `json:"oauth"`
	Filter          swagger.FilterConfig           `json:"filter"`
	SyntaxHighlight *swagger.SyntaxHighlightConfig `json:"syntaxHighlight"`
}

type Configs struct {
	App                 fiber.Config    `json:"app"`
	Swagger             SwaggerConfigs  `json:"swagger"`
	Port                string          `json:"port,omitempty"`
	Handler             handler.Configs `json:"handler,omitempty"`
	AdminEndpointsPath  string          `json:"admin_endpoints_path" config:"ignore"`
	AdminPassword       string          `json:"admin_password" config:"ignore"`
	LogRequestReceive   bool            `json:"log_request_receive"`
	UseTraceId          bool            `json:"use_trace_id"`
	EndpointsPathPrefix string          `json:"endpointsPathPrefix" config:"ignore"`
}
