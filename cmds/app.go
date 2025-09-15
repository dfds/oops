package main

import (
	"log"

	"go.dfds.cloud/bootstrap"
	"go.dfds.cloud/oops/core/config"
	"go.dfds.cloud/oops/core/logging"
	"go.dfds.cloud/oops/feats/api"
	"go.dfds.cloud/oops/feats/jobs"
	"go.uber.org/zap"
)

func main() {
	// setup base
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config")
	}

	builder := bootstrap.Builder()
	builder.EnableLogging(conf.LogDebug, conf.LogLevel)
	builder.EnableHttpRouter(false)
	builder.EnableMetrics()
	builder.EnableOrchestrator("orchestrator")
	manager := builder.Build()
	logging.Logger = manager.Logger
	manager.Orchestrator.Init(logging.Logger)

	logging.Logger.Info("oops launched")

	api.Configure(manager.HttpRouter)

	jobs.Init(manager.Orchestrator)

	<-manager.Context.Done()
	if err := manager.HttpServer.Shutdown(manager.Context); err != nil {
		logging.Logger.Info("HTTP Server was unable to shut down gracefully", zap.Error(err))
	}

	logging.Logger.Info("server shutting down")
}
