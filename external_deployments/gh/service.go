package gh

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/external_deployments"
)

type GithubDeploymentService struct {
	repo          Repository
	ghDeployments []*github.Deployment
}

type DeploymentService interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*external_deployments.Deployment, error)
}

func NewGithubDeploymentService(repo Repository) (*GithubDeploymentService, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo cannot be nil")
	}
	return &GithubDeploymentService{
		repo: repo,
	}, nil
}

func (gs *GithubDeploymentService) ListDeployments(ctx context.Context) ([]*external_deployments.Deployment, error) {
	ghDeployments, err := gs.loadDeployments(ctx)
	if err != nil {
		return nil, err
	}
	return toDeployments(ghDeployments), nil
}

func (gs *GithubDeploymentService) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*external_deployments.Deployment, error) {
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
		successful = append(successful, prevSuccessful)
	}

	populated, err := gs.fillWithCommits(ctx, successful)
	if err != nil {
		return nil, err
	}

	if len(populated) >= 1 {
		inRange = populated[:len(populated)-1]
	} else {
		inRange = populated
	}
	return inRange, nil
}

// loadDeployments loads all deployments on the first time and stores them in cache
func (gs *GithubDeploymentService) loadDeployments(ctx context.Context) ([]*github.Deployment, error) {
	// todo extend cache with newest deployment
	if len(gs.ghDeployments) > 0 {
		return gs.ghDeployments, nil
	}

	var allDeploys []*github.Deployment
	opts := &github.DeploymentsListOptions{
		ListOptions: github.ListOptions{Page: 1},
	}

	for opts.ListOptions.Page > 0 {
		deploys, resp, err := gs.repo.ListDeployments(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("error while fetching github ghDeployments: %w", err)
		}

		allDeploys = append(allDeploys, deploys...)

		if resp.Rate.Remaining <= 10 {
			return nil, fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v",
				resp.Rate.Reset)
		}

		opts.ListOptions.Page = resp.NextPage
	}

	gs.ghDeployments = allDeploys
	return gs.ghDeployments, nil
}

// if none other don't add it (means that this is the first deployment)
func (gs *GithubDeploymentService) findLatestSuccessfulBefore(
	ctx context.Context, deploys []*external_deployments.Deployment, from time.Time) (*external_deployments.Deployment, error) {

	prevDeploys := filterTimerange(deploys, from.Add(-time.Duration(24)*time.Hour), from)
	index := -1
	for i, d := range prevDeploys {
		statuses, err := gs.repo.ListDeploymentStatuses(ctx, d.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
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

func (gs *GithubDeploymentService) fillWithCommits(ctx context.Context, deployments []*external_deployments.Deployment) ([]*external_deployments.Deployment, error) {
	// If there are no deployments to compare it to
	if len(deployments) <= 1 {
		return deployments, nil
	}

	for i, d := range deployments {
		if i+1 >= len(deployments) {
			break
		}

		head := deployments[i].SHA
		base := deployments[i+1].SHA

		commitCmp, err := gs.repo.CompareCommits(ctx, base, head, nil)
		if err != nil {
			return nil, fmt.Errorf("error while comparing commits %w", err)
		}

		switch status := commitCmp.GetStatus(); status {
		case "ahead":
			d.Added = toCommits(commitCmp)

		case "behind":
			d.Removed = toCommits(commitCmp)

		case "diverged":
			d.Added = toCommits(commitCmp)

			mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
			divergedCmp, err := gs.repo.CompareCommits(ctx, mergeBase, base, &github.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("comparing diverged commits: %w", err)
			}
			d.Removed = toCommits(divergedCmp)

		case "identical":
			// No action needed if slices are already nil or empty
		default:
			return nil, fmt.Errorf("unexpected commit status: %s", status)
		}
	}

	return deployments, nil
}

func (gs *GithubDeploymentService) filterSuccessful(ctx context.Context, deployments []*external_deployments.Deployment) ([]*external_deployments.Deployment, error) {
	successful := make([]*external_deployments.Deployment, 0, len(deployments))

	for _, d := range deployments {
		statuses, err := gs.repo.ListDeploymentStatuses(ctx, d.ID, &github.ListOptions{
			PerPage: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
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

func filterTimerange(deployments []*external_deployments.Deployment, from time.Time, to time.Time) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.CreatedAt.After(from) && d.CreatedAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
