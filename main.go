package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v81/github"
	"os"
)

func main() {
	// setup github
	owner := "kemonprogrammer"
	repo := "github-go-client"
	githubPat := os.Getenv("GITHUB_PAT")
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(githubPat)

	deployments, _, _ := client.Repositories.ListDeployments(ctx, owner, repo, &github.DeploymentsListOptions{
		SHA:         "",
		Ref:         "",
		Task:        "",
		Environment: "",
		ListOptions: github.ListOptions{},
	})

	successfulDeployments, err := filterSuccessfulDeployments(client, ctx, owner, repo, deployments)
	if err != nil {
		fmt.Println(err)
	}

	for i, d := range successfulDeployments {
		fmt.Printf("\nDeployment %d:\n", i)

		fmt.Printf("sha: %s\n", d.GetSHA())
		fmt.Printf("id: %d\n", d.GetID())
		fmt.Printf("created at: %s\n", d.GetCreatedAt())
	}

}

func filterSuccessfulDeployments(client *github.Client, ctx context.Context, owner, repo string, deployments []*github.Deployment) ([]*github.Deployment, error) {
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
