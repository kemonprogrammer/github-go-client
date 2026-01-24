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
	repo := "kiali"
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

	for i, d := range deployments {
		fmt.Printf("\nDeployment %d:\n", i)

		fmt.Printf("sha: %s\n", d.GetSHA())
		fmt.Printf("id: %d\n", d.GetID())
		fmt.Printf("created at: %s\n", d.GetCreatedAt())

		statuses, _, _ := client.Repositories.ListDeploymentStatuses(ctx, owner, repo, d.GetID(), &github.ListOptions{
			Page:    0,
			PerPage: 10,
		})
		for _, status := range statuses {
			fmt.Printf("status state: %s\n", status.GetState())
		}
	}

	// read deployments

	//orgs, _, _ := client.Organizations.List(context.Background(), "willnorris", nil)

}
