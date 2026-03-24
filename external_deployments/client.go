package external_deployments

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kemonprogrammer/github-go-client/config"
	"github.com/kemonprogrammer/github-go-client/external_deployments/github"
	"github.com/kemonprogrammer/github-go-client/external_deployments/model"
	"github.com/kemonprogrammer/github-go-client/log"
)

type DeploymentClient interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*model.Deployment, error)
	SetRepo(ctx context.Context, repo string) error
	GetRepo() string
}

func NewDeploymentClient(conf *config.Config) (DeploymentClient, error) {
	if !conf.Enabled {
		return nil, fmt.Errorf("external deployments not enabled")
	}

	provider := conf.Provider
	if provider == "github" {
		owner := conf.Owner
		if len(owner) == 0 {
			return nil, fmt.Errorf("external_service.external_deployments.auth.username not set in config")
		}

		ghAPI, err := github.NewAPI(conf)
		if err != nil {
			return nil, err
		}
		if os.Getenv("TEST") == "true" {
			log.Info("using mock GitHub client")
			ghAPI = github.NewMockAPI()
		}
		return github.NewDeploymentClient(ghAPI)
	}

	return nil, fmt.Errorf("external deployments provider %s not supported ", provider)
}
