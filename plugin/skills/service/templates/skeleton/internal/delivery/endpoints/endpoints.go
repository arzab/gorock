package endpoints

import (
	"github.com/arzab/gorock-kit/rockfiber"

	"__MODULE__/internal/delivery/endpoints/hello"
)

// All lists every HTTP endpoint. The HTTP App registers them as they are.
func All() []rockfiber.FiberEndpoint {
	return []rockfiber.FiberEndpoint{
		hello.Endpoint(),
	}
}
