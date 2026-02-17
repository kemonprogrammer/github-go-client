package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
)

type Repository interface {
	ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error)
	ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error)
	CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}
type GithubRepository struct {
	client                   *github.Client
	name, owner, environment string
}

func NewGithubRepository(client *github.Client, owner, name, environment string) (Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("github client cannot be nil")
	}
	return &GithubRepository{
		client:      client,
		owner:       owner,
		name:        name,
		environment: environment,
	}, nil
}

func (gc *GithubRepository) ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	if opts.Environment == "" {
		opts.Environment = gc.environment
	}
	deploys, resp, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, gc.name, opts)
	return deploys, resp, err
}

func (gc *GithubRepository) ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error) {
	deploys, _, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, gc.name, id, opts)
	return deploys, err
}

func (gc *GithubRepository) CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	deploys, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, gc.name, base, head, opts)
	return deploys, err
}

func (gc *GithubRepository) ListRepositories(ctx context.Context, opts *github.RepositoryListByAuthenticatedUserOptions) ([]*github.Repository, *github.Response, error) {
	repos, resp, err := gc.client.Repositories.ListByAuthenticatedUser(ctx, opts)
	return repos, resp, err
}
