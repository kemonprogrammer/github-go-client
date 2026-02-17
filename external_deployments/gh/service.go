package gh

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/external_deployments"
)

type GithubDeploymentService struct {
	repo                  Repository
	ghDeployments         []*github.Deployment
	successfulDeployments []*external_deployments.Deployment
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
	err := gs.loadDeployments(ctx)

	if err != nil {
		return nil, err
	}
	return toDeployments(gs.ghDeployments), nil
}

// loadSuccessfulDeploymentsInRange
// load deployments
// filter deploys: from after updated at and to is before created at
// if not in successfulDeployments
//   - fetch deployment status
//   - if successful put into successfulDeployments (sort each time)
//
// filter successful deployments in time range
func (gs *GithubDeploymentService) loadSuccessfulDeploymentsInRange(ctx context.Context, from, to time.Time) error {
	if err := gs.loadDeployments(ctx); err != nil {
		return err
	}

	allDeploys := toDeployments(gs.ghDeployments)

	// load successful deployments in Range
	possibleSuccessfulDeploys := filterTimerangeBySuccessPossible(allDeploys, from, to)

	// only refresh success status if not already succeeded
	newPossibleSuccessfulDeploys := make([]*external_deployments.Deployment, 0, len(possibleSuccessfulDeploys))
	for _, possibleDeploy := range possibleSuccessfulDeploys {

		if !slices.ContainsFunc(gs.successfulDeployments, func(deploy *external_deployments.Deployment) bool {
			return deploy.ID == possibleDeploy.ID
		}) {
			newPossibleSuccessfulDeploys = append(newPossibleSuccessfulDeploys, possibleDeploy)
		}
	}

	populated, err := gs.populateSuccessStatus(ctx, newPossibleSuccessfulDeploys)
	if err != nil {
		return err
	}

	newSuccessfulDeploys := append(gs.successfulDeployments, populated...)
	slices.SortFunc(newSuccessfulDeploys, func(a, b *external_deployments.Deployment) int {
		return int(b.SucceededAt.Unix() - a.SucceededAt.Unix()) // assumption: running on 64-bit or higher architecture
	})

	gs.successfulDeployments = newSuccessfulDeploys
	return nil
}

// ListDeploymentsInRange
//
// **Assumption**: list of deployments ordered by succeededAt is append-only, since it deploying an application only adds a deployment status with the current date and not a past date
//
// When creating Deployment statuses manually, the date also can't be set</p>
func (gs *GithubDeploymentService) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*external_deployments.Deployment, error) {

	err := gs.loadSuccessfulDeploymentsInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	successful := gs.successfulDeployments
	fmt.Printf("successful: %d\n", len(successful))

	inRange := filterTimerangeBySucceededAt(successful, from, to)

	// get the one deployment before `from` to compare it to the first inside time range
	// assumption: if there is a successful deployment before `from` it has to have been loaded during `loadSuccessful...InRange`
	// **proof**:
	//  - A1: there has to be one deployment at each time, starting from the first deployment
	//        so combining all succeededAt to updatedAt timespans fills the whole time there was an active deployment
	//  - A2: the updatedAt of an older deployment is always set at the time or after a newer deployment succeeded
	//  - A3: the elements inside gs.successfulDeployments are sorted by succeededAt date from newest to oldest
	//  - defining d := the first successful deployment before `from`
	//    we know that d must have updatedAt after `from`, because of A1 and A2
	//    therefore we know that d also was loaded in `loadSuccessful...InRange`, because of how it's implemented
	//    therefore it must be included in gs.successfulDeployments
	//    because of A3 it must be the first deployment found before `from`

	var oneBefore *external_deployments.Deployment
	for _, sd := range gs.successfulDeployments {
		if sd.SucceededAt.Before(from) {
			oneBefore = sd
			break
		}
	}

	populated, err := gs.fillWithCommits(ctx, append(inRange, oneBefore))
	if err != nil {
		return nil, err
	}

	// remove one before
	if oneBefore != nil {
		populated = populated[:len(populated)-1]
	}
	return populated, nil
}

// loadDeployments loads all deployments on the first time and stores them in cache
func (gs *GithubDeploymentService) loadDeployments(ctx context.Context) error {
	if len(gs.ghDeployments) > 0 {
		prevNewestID := gs.ghDeployments[0].GetID()

		newDeployCount := -1
		var allDeploys []*github.Deployment

		perPage := 100
		opts := &github.DeploymentsListOptions{
			ListOptions: github.ListOptions{Page: 1, PerPage: perPage},
		}

		for opts.ListOptions.Page > 0 && newDeployCount == -1 {
			deploys, resp, err := gs.repo.ListDeployments(ctx, opts)
			if err != nil {
				return fmt.Errorf("error while fetching github ghDeployments: %w", err)
			}

			// search until previous new deployment found
			for i, deploy := range deploys {
				if deploy.GetID() == prevNewestID {
					newDeployCount = i + (opts.ListOptions.Page-1)*perPage
					break
				}
			}
			allDeploys = append(allDeploys, deploys...)

			if resp.Rate.Remaining <= 10 {
				return fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v",
					resp.Rate.Reset)
			}

			opts.ListOptions.Page = resp.NextPage
		}

		if newDeployCount == 0 {
			return nil
		}

		if newDeployCount > 0 {
			// assumption deployments from API are sorted by creation date in descending oder
			newDeploys := allDeploys[:newDeployCount]
			gs.ghDeployments = append(newDeploys, gs.ghDeployments...)
			return nil
		}
		return fmt.Errorf("Could not find cached deployment %d\n", prevNewestID)
	}

	var allDeploys []*github.Deployment
	opts := &github.DeploymentsListOptions{
		ListOptions: github.ListOptions{Page: 1},
	}

	for opts.ListOptions.Page > 0 {
		deploys, resp, err := gs.repo.ListDeployments(ctx, opts)
		if err != nil {
			return fmt.Errorf("error while fetching github ghDeployments: %w", err)
		}

		allDeploys = append(allDeploys, deploys...)

		if resp.Rate.Remaining <= 10 {
			return fmt.Errorf("rate limit nearly exhausted, only 10 calls remaining; resets at %v",
				resp.Rate.Reset)
		}

		opts.ListOptions.Page = resp.NextPage
	}

	gs.ghDeployments = allDeploys
	return nil
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
			behindCmp, err := gs.repo.CompareCommits(ctx, head, base, &github.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("error comparing behind commits: %w", err)
			}
			d.Removed = toCommits(behindCmp)

		case "diverged":
			d.Added = toCommits(commitCmp)

			mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
			divergedCmp, err := gs.repo.CompareCommits(ctx, mergeBase, base, &github.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("error comparing diverged commits: %w", err)
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

func filterTimerangeBySucceededAt(deployments []*external_deployments.Deployment, from time.Time, to time.Time) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.SucceededAt.After(from) && d.SucceededAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func filterTimerangeBySuccessPossible(deployments []*external_deployments.Deployment, from time.Time, to time.Time) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.UpdatedAt.After(from) && d.CreatedAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// populateSuccessStatus assumption: deployment status: x -> success -> inactive
func (gs *GithubDeploymentService) populateSuccessStatus(ctx context.Context, deploys []*external_deployments.Deployment) ([]*external_deployments.Deployment, error) {
	successful := make([]*external_deployments.Deployment, 0, len(deploys))

	for _, d := range deploys {
		// todo get all deployment statuses in case there is a next page
		statuses, err := gs.repo.ListDeploymentStatuses(ctx, d.ID, &github.ListOptions{
			PerPage: 30,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.ID, err)
		}

		for _, status := range statuses {
			if status.GetState() == "success" {
				d.SucceededAt = status.GetUpdatedAt().Time
				successful = append(successful, d)
				break
			}
		}
	}
	return successful, nil
}
