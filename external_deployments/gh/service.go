package gh

import (
	"cmp"
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
	runs                  []*github.WorkflowRun
	runsUpdatedAt         map[int64]time.Time
}

type DeploymentService interface {
	ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*external_deployments.Deployment, error)
}

func NewGithubDeploymentService(repo Repository) (*GithubDeploymentService, error) {
	if repo == nil {
		return nil, fmt.Errorf("repo cannot be nil")
	}
	return &GithubDeploymentService{
		repo:          repo,
		runs:          []*github.WorkflowRun{},
		runsUpdatedAt: map[int64]time.Time{},
	}, nil
}

func (gs *GithubDeploymentService) ListDeployments(ctx context.Context) ([]*external_deployments.Deployment, error) {
	err := gs.loadDeployments(ctx)

	if err != nil {
		return nil, err
	}
	return toDeployments(gs.ghDeployments), nil
}

func (gs *GithubDeploymentService) loadSuccessfulDeployments(ctx context.Context) error {
	if err := gs.loadDeployments(ctx); err != nil {
		return err
	}

	// already cached, so only fetching new successful deploys
	if len(gs.successfulDeployments) > 0 {
		// todo only fetch new successful deploys
		err := gs.updateSuccessfulDeployments(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	// on empty cache, fill cache
	allDeploys := toDeployments(gs.ghDeployments)
	successful, err := gs.populateSuccessStatus(ctx, allDeploys)
	if err != nil {
		return err
	}

	// sort successful deployments from newest to oldest
	succeededAtCmp := func(a, b *external_deployments.Deployment) int {
		return cmp.Compare(b.SucceededAt.Unix(), a.SucceededAt.Unix())
	}
	slices.SortFunc(successful, succeededAtCmp)

	gs.successfulDeployments = successful
	return nil
}

func (gs *GithubDeploymentService) loadWorkflowRuns(ctx context.Context) error {
	runs, _, err := gs.repo.ListWorkflowRuns(ctx, nil)
	if err != nil {
		return err
	}
	gs.runs = runs.WorkflowRuns
	for _, run := range runs.WorkflowRuns {
		gs.runsUpdatedAt[run.GetID()] = run.UpdatedAt.Time
	}
	return nil
}

func (gs *GithubDeploymentService) getUpdatedSHAs() ([]string, error) {

	updatedSHAs := make([]string, 0, len(gs.runs))
	for _, run := range gs.runs {
		if gs.runsUpdatedAt[run.GetID()] != run.GetUpdatedAt().Time {
			updatedSHAs = append(updatedSHAs, run.GetHeadSHA())
		}
	}
	return updatedSHAs, nil
}

func (gs *GithubDeploymentService) updateSuccessfulDeployments(ctx context.Context) error {
	if err := gs.loadWorkflowRuns(ctx); err != nil {
		return err
	}

	// todo run total changed
	updatedSHAs, err := gs.getUpdatedSHAs()
	if err != nil {
		return err
	}

	err = gs.refreshOldDeployments(ctx, updatedSHAs)
	oldDeploysLen := len(gs.ghDeployments)
	if err = gs.loadDeployments(ctx); err != nil {
		return err
	}

	difference := len(gs.ghDeployments) - oldDeploysLen // assumption won't delete deployments
	if difference > 0 {
		additionalDeploys := gs.ghDeployments[:difference]
		additionalSuccessfulDeploys, err := gs.populateSuccessStatus(ctx, toDeployments(additionalDeploys))
		if err != nil {
			return err
		}
		gs.successfulDeployments = append(additionalSuccessfulDeploys, gs.successfulDeployments...)
	}
	return err
}

func (gs *GithubDeploymentService) refreshOldDeployments(ctx context.Context, shas []string) error {

	// todo check
	indices := make([]int, 0, len(gs.successfulDeployments))
	deploysToUpdate := make([]*external_deployments.Deployment, 0, len(gs.successfulDeployments))
	for i, deployment := range gs.successfulDeployments {
		if slices.Contains(shas, deployment.SHA) {
			indices = append(indices, i)
			deploysToUpdate = append(deploysToUpdate, deployment)
		}
	}

	updated, err := gs.populateSuccessStatus(ctx, deploysToUpdate)
	if err != nil {
		return err
	}

	for i, index := range indices {
		gs.successfulDeployments[index] = updated[i]
	}
	return nil
}

// ListDeploymentsInRange
//
// **Assumption**: list of deployments ordered by succeededAt is append-only, since it deploying an application only adds a deployment status with the current date and not a past date
//
// When creating Deployment statuses manually, the date also can't be set</p>
func (gs *GithubDeploymentService) ListDeploymentsInRange(ctx context.Context, from, to time.Time) ([]*external_deployments.Deployment, error) {

	err := gs.loadSuccessfulDeployments(ctx)
	if err != nil {
		return nil, err
	}
	successful := gs.successfulDeployments
	fmt.Printf("successful: %d\n", len(successful))

	inRangeWith1Before := filterTimerangeIncludeNBefore(successful, from, to, 1)

	populated, err := gs.fillWithCommits(ctx, inRangeWith1Before)
	if err != nil {
		return nil, err
	}

	if len(populated) >= 1 {
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
			// assumption deployments from API are sorted by creation date descendingly
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

func filterTimerange(deployments []*external_deployments.Deployment, from time.Time, to time.Time) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.SucceededAt.After(from) && d.SucceededAt.Before(to) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func filterTimerangeIncludeNBefore(deployments []*external_deployments.Deployment, from time.Time, to time.Time, n int) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	beforeIndex := 0
	for i, d := range deployments {
		if d.SucceededAt.After(from) && d.SucceededAt.Before(to) {
			filtered = append(filtered, d)
		}
		if d.SucceededAt.Before(from) {
			beforeIndex = i
			break
		}
	}
	nDeploysBefore := deployments[beforeIndex : beforeIndex+n]
	return append(filtered, nDeploysBefore...)
}

func filterBefore(deployments []*external_deployments.Deployment, time time.Time) []*external_deployments.Deployment {
	filtered := make([]*external_deployments.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.CreatedAt.Before(time) {
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
