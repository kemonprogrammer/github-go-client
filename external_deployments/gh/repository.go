package gh

import (
	"context"
	"fmt"

	"github.com/google/go-github/v81/github"
)

type ClientInterface interface {
	GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error)
	ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error)
	ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error)
	CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}

type GithubRepository struct {
	client             *github.Client
	owner, environment string
}

func NewGithubClient(client *github.Client, owner, environment string) (ClientInterface, error) {
	if client == nil {
		return nil, fmt.Errorf("github client cannot be nil")
	}
	return &GithubRepository{
		client:      client,
		owner:       owner,
		environment: environment,
	}, nil
}

func (gc *GithubRepository) GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error) {
	repo, resp, err := gc.client.Repositories.Get(ctx, gc.owner, repoName)
	if err != nil {
		return nil, nil, err
	}
	return repo, resp, nil
}

func (gc *GithubRepository) ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	fmt.Printf("TRACE ListDeployments\n")
	if opts.Environment == "" {
		opts.Environment = gc.environment
	}
	deploys, resp, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, repoName, opts)
	return deploys, resp, err
}

func (gc *GithubRepository) ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error) {
	fmt.Printf("TRACE ListDeploymentStatuses\n")
	statuses, _, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, repoName, id, opts)
	return statuses, err
}

func (gc *GithubRepository) CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	fmt.Printf("TRACE CompareCommits\n")
	commitCmp, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, repoName, base, head, opts)
	if err != nil {
		return nil, err
	}

	return commitCmp, err
}
