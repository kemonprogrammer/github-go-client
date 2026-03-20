package external_deployments

import (
	"context"
	"fmt"
	"time"

	"github.com/kemonprogrammer/github-go-client/config"
	"github.com/kemonprogrammer/github-go-client/external_deployments/gh"
	"github.com/kemonprogrammer/github-go-client/external_deployments/model"
)

type DeploymentService interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error)
	ValidateRepo(ctx context.Context) error
}

func NewDeploymentService(cfg *config.Config, repo string) (DeploymentService, error) {
	if cfg.Enabled == true {
		if cfg.Provider == "github" {
			deploymentClient := gh.MakeGithubClientInterface(cfg)
			return gh.NewGithubDeploymentService(deploymentClient, repo)
		}

		return nil, fmt.Errorf("external deployments provider %s not supported ", cfg.Provider)
	}
	return nil, fmt.Errorf("external deployments not enabled")
}
