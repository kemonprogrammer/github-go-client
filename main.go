package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/deployment"
	"github.com/kemonprogrammer/github-go-client/gh"
)

type Response struct {
	deployments []deployment.Deployment
}

type Params struct {
	From, To time.Time
}

func fillParams(from, to string) (*Params, error) {
	//dateTimeFormat := "2006-01-02T00:00:00Z"
	dateFrom, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse date from %s, %w", from, err)
	}
	dateTo, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse date to %s, %w", from, err)
	}
	params := &Params{
		From: dateFrom,
		To:   dateTo,
	}
	//fmt.Printf("params: %v", params)
	return params, nil
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
	queryFrom := os.Getenv("FROM")
	queryTo := os.Getenv("TO")

	user, resp, err := client.Users.Get(context.Background(), "")
	if err == nil {
		// GitHub returns the version used in the 'X-GitHub-Api-Version' header
		actualVersion := resp.Header.Get("X-GitHub-Api-Version")
		fmt.Printf("user: %s\n", *user.Login)
		fmt.Printf("API Version used by server: %s\n", actualVersion)
	}

	params, err := fillParams(queryFrom, queryTo)
	if err != nil {
		fmt.Println(err)
		return
	}

	ghRepo, err := gh.NewGithubRepository(client, owner, repo, env)
	if err != nil {
		fmt.Println(err)
		return
	}
	deploymentService, err := gh.NewService(ghRepo)
	if err != nil {
		fmt.Println(err)
		return
	}

	// todo what can be cached // how to refresh cache
	// notes: older deployments can't be updated, only deleted
	deployments, err := deploymentService.ListDeploymentsInRange(ctx, params.From, params.To)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("len deploys: %d\n", len(deployments))

	fmt.Printf("deployments response: %+v", deployments)
}
