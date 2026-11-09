package jobs

import (
	"context"
	"time"

	"github.com/arzab/gorock-kit/rockcron"

	"github.com/yourname/taskapi/pkg/services"
)

// All lists every scheduled job. The cron App registers them.
func All() []rockcron.Job {
	return []rockcron.Job{
		rockcron.Every("ping-db", time.Minute, pingDB),
	}
}

func pingDB(ctx context.Context) error {
	return services.Repository().Ping(ctx)
}
