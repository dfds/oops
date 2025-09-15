package jobs

import (
	"context"

	"go.dfds.cloud/oops/core/logging"
	"go.dfds.cloud/oops/feats/jobs/handlers"
	"go.dfds.cloud/orchestrator"
)

func Init(orc *orchestrator.Orchestrator) {
	configPrefix := "SSU_OOPS_JOB"

	orc.AddJob(configPrefix, orchestrator.NewJob("dummy", func(ctx context.Context) error {
		logging.Logger.Info("dummy")
		return nil
	}), &orchestrator.Schedule{})

	orc.AddJob(configPrefix, orchestrator.NewJob("route53Backup", handlers.Route53Backup), &orchestrator.Schedule{})

	orc.Run()
}
