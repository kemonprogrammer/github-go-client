package githubClient

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/deployment"
)

type GithubService struct {
	Repo          deployment.Repo
	Client        *github.Client
	Context       context.Context
	ghDeployments []*github.Deployment
}

func (gs *GithubService) loadDeployments() ([]*github.Deployment, error) {
	// todo extend cache with time range
	if len(gs.ghDeployments) > 0 {
		return gs.ghDeployments, nil
	}

	ghDeployments, _, err := gs.Client.Repositories.ListDeployments(
		gs.Context, gs.Repo.Owner, gs.Repo.Name, &github.DeploymentsListOptions{
			Environment: gs.Repo.Environment,
			//SHA:         "",
			//Ref:         "",
			//Task:        "",
			//ListOptions: github.ListOptions{}, // todo handle more than 30 ghDeployments (default)
		})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("error while fetching github ghDeployments: %s", err)
	}
	gs.ghDeployments = ghDeployments
	return gs.ghDeployments, nil
}

func (gs *GithubService) ListDeployments() ([]*deployment.Deployment, error) {
	ghDeployments, err := gs.loadDeployments()
	if err != nil {
		return nil, err
	}
	return toDeployments(ghDeployments), nil
}

func (gs *GithubService) ListDeploymentsInRange(from, to time.Time) ([]*deployment.Deployment, error) {
	// todo handle deployment not in range
	ghDeployments, err := gs.loadDeployments()
	if err != nil {
		return nil, err
	}
	deployments := toDeployments(ghDeployments)

	inTimeDeploys := filterTimerange(deployments, from, to)
	successfulDeploys, err := gs.filterSuccessful(inTimeDeploys)
	if err != nil {
		return nil, err
	}

	return successfulDeploys, nil
}

func (gs *GithubService) FillWithCommits(deployments []*deployment.Deployment) error {
	for i, d := range deployments {
		fmt.Printf("\nDeployment %d:\n", d.GetID())

		fmt.Printf("sha: %s\n", d.GetSHA())
		fmt.Printf("created at: %s\n", d.GetCreatedAt())

		if len(deployments) < 2 || i+1 >= len(deployments) {
			continue
		}
		head := deployments[i].GetSHA()
		base := deployments[i+1].GetSHA()

		commitCmp, _, err := gs.Client.Repositories.CompareCommits(gs.Context, gs.Repo.Owner, gs.Repo.Name, base, head, &github.ListOptions{
			// todo handle more than 30 commits  (default) -> maybe "<first 7 commits> 24 more commits\n<compare-url>"
		})

		if err != nil {
			return fmt.Errorf("error while comparing commits %s", err)
		}

		// to string method
		for _, c := range commitCmp.Commits {
			fmt.Printf("+ %s\n", deployment.GetTitle(c.Commit.GetMessage()))
		}

		// * pseudo *
		// if status == "ahead" skip

		// else if status == "diverged"
		// 1. add commits to addedCommits
		// 1. call compareCommits() with mergeBaseSha...baseSha
		// 1. add commits to removedCommits

		// else if status == "behind"
		// 1. add all commits to removedCommits
		if commitCmp.GetStatus() == "ahead" {
			d.CommitsAdded = toCommits(commitCmp)
			d.CommitsRemoved = make([]deployment.Commit, 0)

		} else if commitCmp.GetStatus() == "behind" {
			d.CommitsAdded = make([]deployment.Commit, 0)
			d.CommitsRemoved = toCommits(commitCmp)

		} else if commitCmp.GetStatus() == "diverged" {
			d.CommitsAdded = toCommits(commitCmp)
			mergeBase := commitCmp.GetMergeBaseCommit().GetSHA()
			divergedCommitCmp, _, err := gs.Client.Repositories.CompareCommits(gs.Context, gs.Repo.Owner, gs.Repo.Name, mergeBase, base, &github.ListOptions{})
			if err != nil {
				return fmt.Errorf("error while comparing commits %s", err)
			}
			d.CommitsRemoved = toCommits(divergedCommitCmp)

		} else if commitCmp.GetStatus() == "identical" {
			d.CommitsAdded = make([]deployment.Commit, 0)
			d.CommitsRemoved = make([]deployment.Commit, 0)
		}
	}

	return nil
}

func toCommits(commitCmp *github.CommitsComparison) []deployment.Commit {
	commits := make([]deployment.Commit, commitCmp.GetTotalCommits())
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

func toCommit(commit *github.RepositoryCommit) deployment.Commit {
	return deployment.Commit{
		SHA:   commit.GetSHA(), // sha somehow stored in commit, now commit.Commit
		Title: commit.Commit.GetMessage(),
		URL:   commit.Commit.GetURL(),
	}
}

func (gs *GithubService) filterSuccessful(deployments []*deployment.Deployment) ([]*deployment.Deployment, error) {
	result := make([]*deployment.Deployment, 0, len(deployments))

	for _, d := range deployments {
		if d.ID == nil {
			continue
		}

		statuses, _, err := gs.Client.Repositories.ListDeploymentStatuses(gs.Context, gs.Repo.Owner, gs.Repo.Name, d.GetID(), &github.ListOptions{
			PerPage: 10,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get deployment statuses for %d: %w", d.GetID(), err)
		}
		for _, status := range statuses {
			if status.GetState() == "success" {
				result = append(result, d)
				break
			}
		}
	}
	return result, nil
}

func filterTimerange(deployments []*deployment.Deployment, from time.Time, to time.Time) []*deployment.Deployment {
	filteredDeploys := make([]*deployment.Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.GetCreatedAt().After(from) && d.GetCreatedAt().Before(to) {
			filteredDeploys = append(filteredDeploys, d)
		}
	}
	return filteredDeploys
}
