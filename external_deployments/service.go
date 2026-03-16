package external_deployments

import (
	"context"
	"time"

	"github.com/kemonprogrammer/github-go-client/external_deployments/types"
)

type DeploymentService interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*types.Deployment, error)
	ValidateRepo(ctx context.Context) error
}
