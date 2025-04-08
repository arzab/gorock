package endpoints

import (
	"github.com/arzab/gorock/pkg/http/endpoints"
	"github.com/arzab/gorock/delivery/endpoints/ga"
	//Imports
)

func HttpEndpoints() []endpoints.FiberEndpoint {
	result := make([]endpoints.FiberEndpoint, 0)

	result = append(result, ga.Endpoint())
	//Endpoints

	return result
}
