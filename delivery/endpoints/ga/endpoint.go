package ga

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
// @Success 200 {object} Response "Response"
// @Failure default {object} responses.ErrorResponse
// @Router /hello [get]
func Endpoint() endpoints.FiberEndpoint {
	return endpoints.BuildFiberEndpoint(
		"get",
		"/hello",
		handlers(),
	)
}
