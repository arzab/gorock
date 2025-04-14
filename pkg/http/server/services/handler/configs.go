package handler

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

type Configs struct {
	IgnoreLogError                 bool            `json:"ignore_log_error" config:"ignore"`
	MaskInternalServerErrorMessage string          `json:"mask_internal_server_error_message" config:"ignore"`
	MonitoringConfigs              *monitor.Config `json:"monitoring_configs" config:"ignore"`
	CorsConfigs                    *cors.Config    `json:"cors" config:"ignore"`
}
