package handlers

import (
	"context"

	"go.dfds.cloud/oops/core/logging"
)

func Route53Backup(ctx context.Context) error {
	logging.Logger.Info("Taking backup of Route53 zones")

	return nil
}
