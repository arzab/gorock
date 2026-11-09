package endpoints

import (
	"github.com/arzab/gorock-kit/rockfiber"

	"github.com/yourname/taskapi/internal/delivery/endpoints/create_task"
	"github.com/yourname/taskapi/internal/delivery/endpoints/get_task"
)

// All lists every HTTP endpoint. The HTTP App registers them as they are.
func All() []rockfiber.FiberEndpoint {
	return []rockfiber.FiberEndpoint{
		create_task.Endpoint(),
		get_task.Endpoint(),
	}
}
