package githubClient

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/deployment"
)

type GithubService struct {
	Client        RepositoryClient
	ghDeployments []*github.Deployment
}

type RepositoryClient interface {
	ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, error)
	ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error)
	CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error)
}

// maybe rename
type GithubRepositoryClient struct {
	Client                   *github.Client
	Repo, Owner, Environment string
}

func (gc *GithubRepositoryClient) ListDeployments(ctx context.Context, opts *github.DeploymentsListOptions) ([]*github.Deployment, error) {
	if opts == nil {
		opts = &github.DeploymentsListOptions{
			Environment: gc.Environment,
			//SHA:         "",
			//Ref:         "",
			//Task:        "",
			//ListOptions: github.ListOptions{}, // todo handle more than 30 ghDeployments (default)
		}
	}
	deploys, _, err := gc.Client.Repositories.ListDeployments(ctx, gc.Owner, gc.Repo, opts)
	return deploys, err
}

func (gc *GithubRepositoryClient) ListDeploymentStatuses(ctx context.Context, id int64, opts *github.ListOptions) ([]*github.DeploymentStatus, error) {
	deploys, _, err := gc.Client.Repositories.ListDeploymentStatuses(ctx, gc.Owner, gc.Repo, id, opts)
	return deploys, err
}

func (gc *GithubRepositoryClient) CompareCommits(ctx context.Context, base, head string, opts *github.ListOptions) (*github.CommitsComparison, error) {
	if opts == nil {
		opts = &github.ListOptions{
			// todo handle more than 30 commits  (default) -> maybe "<first 7 commits> 24 more commits\n<compare-url>"
		}
	}
	deploys, _, err := gc.Client.Repositories.CompareCommits(ctx, gc.Owner, gc.Repo, base, head, opts)
	return deploys, err
}

func (gs *GithubService) loadDeployments(ctx context.Context) ([]*github.Deployment, error) {
	// todo extend cache with time range
	if len(gs.ghDeployments) > 0 {
		return gs.ghDeployments, nil
	}

	ghDeployments, err := gs.Client.ListDeployments(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error while fetching github ghDeployments: %w", err)
	}
	gs.ghDeployments = ghDeployments
	return gs.ghDeployments, nil
}

func (gs *GithubService) ListDeployments(ctx context.Context) ([]*deployment.Deployment, error) {
	ghDeployments, err := gs.loadDeployments(ctx)
	if err != nil {
		return nil, err
	}
	return toDeployments(ghDeployments), nil
}

func (gs *GithubService) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*deployment.Deployment, error) {
	ghDeployments, err := gs.loadDeployments(ctx)
	if err != nil {
		return nil, err
	}
	allDeploys := toDeployments(ghDeployments)

	inRange := filterTimerange(allDeploys, from, to)
	successful, err := gs.filterSuccessful(ctx, inRange)
	if err != nil {
		return nil, err
	}

	// fetch one deployment before first in time range to compare it to first in deployment in time range
	prevSuccessful, err := gs.findLatestSuccessfulBefore(ctx, allDeploys, from)
	if err != nil {
		return nil, err
	}

	if prevSuccessful != nil {
		return append(successful, prevSuccessful), nil
	}
	return successful, nil
}

// todo repeat load previous Deployments until index = -1
// if none other don't add it (means that this is the first deployment)
func (gs *GithubService) findLatestSuccessfulBefore(
	ctx context.Context, deploys []*deployment.Deployment, from time.Time) (*deployment.Deployment, error) {

	prevDeploys := filterTimerange(deploys, from.Add(-time.Duration(24)*time.Hour), from)
	index := -1
	for i, d := range prevDeploys {
		statuses, err := gs.Client.ListDeploymentStatuses(ctx, d.GetID(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.GetID(), err)
		}

		for _, status := range statuses {
			if status.GetState() == "success" {
				index = i
				break
			}
		}
	}
	if index == -1 {
		return nil, nil
	}
	return prevDeploys[index], nil
}

func (gs *GithubService) FillWithCommits(ctx context.Context, deployments []*deployment.Deployment) error {
	for i, d := range deployments {
		if len(deployments) < 2 || i+1 >= len(deployments) {
			continue
		}

		head := deployments[i].GetSHA()
		base := deployments[i+1].GetSHA()

		commitCmp, err := gs.Client.CompareCommits(ctx, base, head, nil)

		if err != nil {
			return fmt.Errorf("error while comparing commits %w", err)
		}

		switch status := commitCmp.GetStatus(); status {
		case "ahead":
			d.CommitsAdded = toCommits(commitCmp)

		case "behind":
			d.CommitsRemoved = toCommits(commitCmp)

		case "diverged":
			d.CommitsAdded = toCommits(commitCmp)

			mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
			divergedCmp, err := gs.Client.CompareCommits(ctx, mergeBase, base, &github.ListOptions{})
			if err != nil {
				return fmt.Errorf("comparing diverged commits: %w", err)
			}
			d.CommitsRemoved = toCommits(divergedCmp)

		case "identical":
			// No action needed if slices are already nil or empty
		default:
			return fmt.Errorf("unexpected commit status: %s", status)
		}
	}

	return nil
}

func toCommits(commitCmp *github.CommitsComparison) []*deployment.Commit {
	commits := make([]*deployment.Commit, commitCmp.GetTotalCommits())
	for i, commit := range commitCmp.Commits {
		commits[i] = toCommit(commit)
	}
	return commits
}

func toDeployment(githubDeploy *github.Deployment) *deployment.Deployment {
	if githubDeploy == nil {
		return nil
	}
	return &deployment.Deployment{
		ID:             githubDeploy.ID,
		DeploymentUrl:  githubDeploy.URL,
		SHA:            githubDeploy.SHA,
		CreatedAt:      githubDeploy.CreatedAt.GetTime(),
		UpdatedAt:      githubDeploy.UpdatedAt.GetTime(),
		ComparisonUrl:  nil,
		CommitsAdded:   nil,
		CommitsRemoved: nil,
	}
}

func toDeployments(ghDeployments []*github.Deployment) []*deployment.Deployment {
	deployments := make([]*deployment.Deployment, len(ghDeployments))
	for i, d := range ghDeployments {
		deployments[i] = toDeployment(d)
	}
	return deployments
}

func toCommit(commit *github.RepositoryCommit) *deployment.Commit {
	return &deployment.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, now commit.Commit
		Title: commit.Commit.GetMessage(),
		URL:   commit.Commit.GetURL(),
	}
}

func (gs *GithubService) filterSuccessful(ctx context.Context, deployments []*deployment.Deployment) ([]*deployment.Deployment, error) {
	successful := make([]*deployment.Deployment, 0, len(deployments))

	for _, d := range deployments {
		if d.ID == nil {
			continue
		}

		statuses, err := gs.Client.ListDeploymentStatuses(ctx, d.GetID(), &github.ListOptions{
			PerPage: 10,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.GetID(), err)
		}
		for _, status := range statuses {
			if status.GetState() == "success" {
				successful = append(successful, d)
				break
			}
		}
	}
	return successful, nil
}

func filterTimerange(deployments []*deployment.Deployment, from time.Time, to time.Time) []*deployment.Deployment {
	filtered := make([]*deployment.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.GetCreatedAt().After(from) && d.GetCreatedAt().Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
