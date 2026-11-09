package create_task

import "github.com/arzab/gorock-kit/rockfiber"

// Endpoint describes itself: method, path and the handler chain.
func Endpoint() rockfiber.FiberEndpoint {
	return rockfiber.POST("/tasks", handlers()...)
}
