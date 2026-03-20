package gh

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/config"
)

// GithubClientInterface mock for testing
type GithubClientInterface interface {
	GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error)
	ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error)
	ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error)
	CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}

type GithubClient struct {
	client             *github.Client
	owner, environment string
}

func MakeGithubClientInterface(cfg *config.Config) GithubClientInterface {
	githubPat := cfg.Token
	gh := github.NewClient(nil).WithAuthToken(githubPat)
	clientInterface, err := NewGithubClient(gh, cfg.Owner, cfg.Env)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return clientInterface
}

func NewGithubClient(client *github.Client, owner, environment string) (GithubClientInterface, error) {
	if client == nil {
		return nil, fmt.Errorf("github client cannot be nil")
	}
	return &GithubClient{
		client:      client,
		owner:       owner,
		environment: environment,
	}, nil
}

func (gc *GithubClient) GetRepository(ctx context.Context, repoName string) (*github.Repository, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Printf("TRACE GetRepository took %v\n", time.Since(start))
	}()
	repo, resp, err := gc.client.Repositories.Get(ctx, gc.owner, repoName)
	if err != nil {
		return nil, nil, err
	}
	return repo, resp, nil
}

func (gc *GithubClient) ListDeployments(ctx context.Context, repoName string, opts *github.DeploymentsListOptions) ([]*github.Deployment, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Printf("TRACE ListDeployments took %v\n", time.Since(start))
	}()
	if opts.Environment == "" {
		opts.Environment = gc.environment
	}
	deploys, resp, err := gc.client.Repositories.ListDeployments(ctx, gc.owner, repoName, opts)
	return deploys, resp, err
}

func (gc *GithubClient) ListDeploymentStatuses(ctx context.Context, repoName string, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, *github.Response, error) {
	start := time.Now()
	defer func() {
		log.Printf("TRACE ListDeploymentStatuses took %v\n", time.Since(start))
	}()
	statuses, resp, err := gc.client.Repositories.ListDeploymentStatuses(ctx, gc.owner, repoName, id, opts)
	return statuses, resp, err
}

func (gc *GithubClient) CompareCommits(ctx context.Context, repoName, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	start := time.Now()
	defer func() {
		log.Printf("TRACE CompareCommits took %v\n", time.Since(start))
	}()
	commitCmp, _, err := gc.client.Repositories.CompareCommits(ctx, gc.owner, repoName, base, head, opts)
	if err != nil {
		return nil, err
	}

	return commitCmp, err
}
