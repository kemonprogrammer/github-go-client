package main

import (
	"context"
	"errors"
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

	owner := os.Getenv("OWNER")

	// params
	workload := os.Getenv("WORKLOAD")
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		fmt.Println(err)
		return
	}

	// GitHub returns the version used in the 'X-GitHub-Api-Version' header
	actualVersion := resp.Header.Get("X-GitHub-Api-Version")
	fmt.Printf("user: %s\n", *user.Login)
	fmt.Printf("API Version used by server: %s\n", actualVersion)

	repoService := NewRepoService(client, user.GetLogin())

	repo, err := repoService.findRepoFromWorkload(ctx, workload)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Repo found: %s\n", repo)

	//repo := "github-go-client"
	queryFrom := os.Getenv("FROM")
	queryTo := os.Getenv("TO")

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

func (rs *RepoService) findRepoFromWorkload(ctx context.Context, workload string) (string, error) {
	repoName := extractRepoName(workload)
	allRepos := make([]*github.Repository, 0)

	page := 1

	for page > 0 {
		repos, resp, err := rs.client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
			Visibility:  "all",
			Affiliation: "",
			Type:        "",
			Sort:        "",
			Direction:   "",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 30,
			},
		})
		if err != nil {
			return "", err
		}

		allRepos = append(allRepos, repos...)

		fmt.Printf("page %d\n", page)
		for _, repo := range allRepos {
			fmt.Printf("repo: %s\n", repo.GetName())
		}

		page = resp.NextPage
	}

	for _, repo := range allRepos {
		if repo.GetName() == repoName {
			return repo.GetName(), nil
		}
	}

	return "", errors.New("No repo found for workload " + workload)
}

type RepoService struct {
	client *github.Client
	user   string
}

func NewRepoService(client *github.Client, user string) RepoService {
	return RepoService{
		client: client,
		user:   user,
	}
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
