package endpoints

import (
	"github.com/arzab/gorock/delivery/endpoints/asf"
	"github.com/arzab/gorock/delivery/endpoints/dasf"
	"github.com/arzab/gorock/pkg/http/endpoints"
)

func HttpEndpoints() []endpoints.FiberEndpoint {
	result := make([]endpoints.FiberEndpoint, 0)

	//Endpoints
	result = append(result, asf.Endpoint())
	result = append(result, dasf.Endpoint())
	//Endpoints

	return result
}
