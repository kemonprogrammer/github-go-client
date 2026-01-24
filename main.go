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

	for i, d := range filterSuccessfulDeployments(client, ctx, owner, repo, deployments) {
		fmt.Printf("\nDeployment %d:\n", i)

		fmt.Printf("sha: %s\n", d.GetSHA())
		fmt.Printf("id: %d\n", d.GetID())
		fmt.Printf("created at: %s\n", d.GetCreatedAt())
	}

	// read deployments

	//orgs, _, _ := client.Organizations.List(context.Background(), "willnorris", nil)
}

func filterSuccessfulDeployments(client *github.Client, ctx context.Context, owner string, repo string, deployments []*github.Deployment) []*github.Deployment {
	var result []*github.Deployment

	for _, d := range deployments {

		statuses, _, _ := client.Repositories.ListDeploymentStatuses(ctx, owner, repo, d.GetID(), &github.ListOptions{
			Page:    0,
			PerPage: 10,
		})
		for _, status := range statuses {
			if status.GetState() == "success" {
				result = append(result, d)
			}
		}
	}
	return result
}
