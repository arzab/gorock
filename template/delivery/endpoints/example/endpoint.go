package example

import (
	"github.com/arzab/gorock/pkg/http/endpoints"
	_ "github.com/arzab/gorock/pkg/http/responses"
)

// Endpoint Definition
// @Summary Summary
// @Description Description
// @Tags Tag
// @Produce     json
// @Accept 		json
// @Param 		body body  Params true "comment"
// @Success 200 {integer} int "Кол-во пользователей"
// @Failure default {object} responses.ErrorResponse
// @Router /{path} [{method}]
func Endpoint() endpoints.FiberEndpoint {
	return endpoints.BuildFiberEndpoint(
		"{method}",
		"/{path}",
		handlers(),
	)
}
