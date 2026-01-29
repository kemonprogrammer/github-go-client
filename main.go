package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v81/github"
	"os"
	"strings"
)

type Response struct {
	deployments []Deployment
}

type Deployment struct {
	tmpD github.Deployment
	tmpC github.Commit

	// deployment
	ID            *int64            `json:"id,omitempty"`
	DeploymentUrl *string           `json:"deployment_url,omitempty"`
	SHA           *string           `json:"sha,omitempty"`
	CreatedAt     *github.Timestamp `json:"created_at,omitempty"`
	UpdatedAt     *github.Timestamp `json:"updated_at,omitempty"`

	// commits
	ComparisonUrl  *string
	CommitsAdded   []DeploymentCommit
	CommitsRemoved []DeploymentCommit
}

type DeploymentCommit struct {
	SHA   *string `json:"sha,omitempty"`
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

func main() {
	if strings.ToUpper(os.Getenv("TEST")) == "TRUE" {
		return
	}

	// setup github
	owner := os.Getenv("OWNER")
	repo := "github-go-client"
	githubPat := os.Getenv("GITHUB_PAT")
	env := os.Getenv("ENVIRONMENT")
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(githubPat)

	// todo what can be cached // how to refresh cache
	// notes: older deployments can't be updated, only deleted
	deployments, _, _ := client.Repositories.ListDeployments(ctx, owner, repo, &github.DeploymentsListOptions{
		SHA:         "",
		Ref:         "",
		Task:        "",
		Environment: env,
		ListOptions: github.ListOptions{}, // todo handle more than 30 deployments (default)
	})
	fmt.Printf("len deploys: %d", len(deployments))

	successfulDeployments, err := FilterSuccessful(client, ctx, owner, repo, deployments)
	if err != nil {
		fmt.Println(err)
	}

	for i, d := range successfulDeployments {
		fmt.Printf("\nDeployment %d:\n", d.GetID())

		fmt.Printf("sha: %s\n", d.GetSHA())
		fmt.Printf("created at: %s\n", d.GetCreatedAt())

		if len(successfulDeployments) < 2 || i+1 >= len(successfulDeployments) {
			continue
		}
		head := successfulDeployments[i].GetSHA()
		base := successfulDeployments[i+1].GetSHA()

		commitCmp, _, err := client.Repositories.CompareCommits(ctx, owner, repo, base, head, &github.ListOptions{
			Page:    0,
			PerPage: 10, // todo handle more than 10 commits -> maybe "61 more commits\n<compare-url>"
		})

		if err != nil {
			fmt.Println(err)
			return
		}

		for _, c := range commitCmp.Commits {
			fmt.Printf("+ %s\n", GetTitle(c.Commit.GetMessage()))
		}
	}
}

func GetTitle(message string) string {
	return strings.Split(message, "\n")[0]
}

func FilterSuccessful(client *github.Client, ctx context.Context, owner, repo string, deployments []*github.Deployment) ([]*github.Deployment, error) {
	result := make([]*github.Deployment, 0, len(deployments))

	for _, d := range deployments {
		if d.ID == nil {
			continue
		}

		statuses, _, err := client.Repositories.ListDeploymentStatuses(ctx, owner, repo, d.GetID(), &github.ListOptions{
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
