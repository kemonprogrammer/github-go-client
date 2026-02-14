package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/kemonprogrammer/github-go-client/external_deployments"
	"github.com/kemonprogrammer/github-go-client/external_deployments/gh"
)

type Response struct {
	deployments []external_deployments.Deployment
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
	githubPat := os.Getenv("GITHUB_PAT")
	env := os.Getenv("ENVIRONMENT")
	ctx := context.Background()
	client := github.NewClient(nil).WithAuthToken(githubPat)

	// params
	workload := os.Getenv("WORKLOAD")
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		fmt.Println(err)
		return
	}
	owner := user.GetLogin()
	repoName := extractRepoName(workload)
	_, _, err = client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		fmt.Println(err)
		fmt.Println(fmt.Errorf("no repository found for workload %s", workload))
	}

	// GitHub returns the version used in the 'X-GitHub-Api-Version' header
	fmt.Printf("user: %s\n", *user.Login)

	queryFrom := os.Getenv("FROM")
	queryTo := os.Getenv("TO")

	params, err := fillParams(queryFrom, queryTo)
	if err != nil {
		fmt.Println(err)
		return
	}

	ghRepo, err := gh.NewGithubRepository(client, owner, repoName, env)
	if err != nil {
		fmt.Println(err)
		return
	}
	deploymentService, err := gh.NewGithubDeploymentService(ghRepo)
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

func extractRepoName(workload string) string {
	regexStr := "-v\\d.*"
	r, err := regexp.Compile(regexStr)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	match, _ := regexp.MatchString(regexStr, workload)
	repoName := workload
	if match {
		repoName = r.ReplaceAllString(workload, "")
	}
	return repoName
}
