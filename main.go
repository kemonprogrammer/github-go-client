package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/deployment"
	"github.com/kemonprogrammer/github-go-client/githubClient"
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
		return nil, fmt.Errorf("couldn't parse date from %s, %s", from, err)
	}
	dateTo, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse date to %s, %s", from, err)
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

	params, err := fillParams(queryFrom, queryTo)

	//t, _ :=time.Parse("2025-09-11", "2026-01-29")
	//ghT := github.Timestamp{ t}

	if err != nil {
		fmt.Println(err)
		return
	}

	deploymentService := githubClient.GithubService{
		Client: client,
		Repo: deployment.Repo{
			Owner:       owner,
			Name:        repo,
			Environment: env,
		},
		Context: ctx,
	}

	// todo what can be cached // how to refresh cache
	// notes: older deployments can't be updated, only deleted
	deployments, err := deploymentService.ListDeploymentsInRange(params.From, params.To)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("len deploys: %d", len(deployments))

	err = deploymentService.FillWithCommits(deployments)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("\ndeployments response: %+v", deployments)
}

type DeploymentService interface {

	//	todo add cache
}
