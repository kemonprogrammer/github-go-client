package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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

func loadExampleRunsCache() (map[int64]time.Time, error) {
	list := []string{
		"22008656798", "2026-02-14T15:43:15Z",
		"22007681786", "2026-02-14T00:35:29Z",
		"21989245212", "2026-02-13T13:50:41Z",
		"21749262779", "2026-02-06T11:40:34Z",
		"21749095244", "2026-02-06T11:34:19Z",
		"21565595496", "2026-02-01T15:40:51Z",
		"21565343898", "2026-02-04T11:54:43Z",
		"21554611557", "2026-02-04T11:53:38Z",
		"21500146540", "2026-01-31T15:43:03Z",
		"21488004493", "2026-01-29T17:22:20Z",
		"21487587270", "2026-01-29T17:09:34Z",
		"21487159087", "2026-01-29T23:04:49Z",
		"21487119192", "2026-01-29T16:56:16Z",
		"21324259007", "2026-01-25T00:44:04Z",
		"21324180293", "2026-01-25T00:37:53Z",
		"21323251674", "2026-01-24T23:20:39Z",
		"21322950116", "2026-01-24T22:55:56Z",
		"21318148380", "2026-01-24T16:33:41Z",
		"21317603428", "2026-01-24T15:48:57Z",
		"21317530527", "2026-01-24T15:43:14Z",
		"21317461823", "2026-01-24T15:38:08Z",
		"21317373606", "2026-01-24T15:30:38Z",
	}
	cache := make(map[int64]time.Time, len(list)/2)
	for i, _ := range list {
		if i%2 == 1 {
			key, err := strconv.ParseInt(list[i-1], 10, 64)
			if err != nil {
				return nil, err
			}
			val, err := time.Parse(time.RFC3339, list[i])
			if err != nil {
				return nil, err
			}
			cache[key] = val
		}
	}
	return cache, nil
}

func main() {

	if strings.ToUpper(os.Getenv("TEST")) == "TRUE" {
		cache, err := loadExampleRunsCache()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(cache)
		return
	}
	cfg := SetupConfig()
	deploymentClient := MakeClient(cfg)

	workload := os.Getenv("WORKLOAD")

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := httpHandler(context.Background(), cfg, workload, deploymentClient)
		if err != nil {
			fmt.Println(err)
			return
		}
		newerDeployments := resp.Deployments
		fmt.Printf("len newer deploys: %d\n", len(newerDeployments))

		fmt.Printf("newer deployments response: %+v\n", newerDeployments)
	}()

	// -- 2nd call
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := httpHandler(context.Background(), cfg, workload, deploymentClient)
		if err != nil {
			fmt.Println(err)
			return
		}
		newerDeployments := resp.Deployments
		fmt.Printf("2nd time to test cache: deployments response: %+v\n", newerDeployments)
	}()

	wg.Wait()
}

func SetupConfig() *Config {
	// setup github
	return &Config{
		owner:    os.Getenv("OWNER"),
		env:      os.Getenv("ENVIRONMENT"),
		token:    os.Getenv("GITHUB_PAT"),
		enabled:  true,
		provider: "github",
	}
}

func MakeClient(cfg *Config) gh.ClientInterface {
	githubPat := cfg.token
	client := github.NewClient(nil).WithAuthToken(githubPat)
	ghRepo, err := gh.NewGithubClient(client, cfg.owner, cfg.env)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return ghRepo
}

type Config struct {
	enabled  bool
	provider string
	owner    string
	env      string
	token    string
}

func NewDeploymentService(cfg *Config, clientInterface gh.ClientInterface, repo string) (gh.DeploymentService, error) {
	if cfg.enabled == true {
		if cfg.provider == "github" {
			return gh.NewGithubDeploymentService(clientInterface, repo)
		}

		return nil, fmt.Errorf("external deployments provider %s not supported ", cfg.provider)
	}
	return nil, fmt.Errorf("external deployments not enabled")
}

type DeploymentResponse struct {
	Deployments []*external_deployments.Deployment `json:"deployments"`
}

func httpHandler(ctx context.Context, cfg *Config, workload string, deploymentClient gh.ClientInterface) (*DeploymentResponse, error) {
	repo := extractRepoName(workload)
	deploymentService, err := NewDeploymentService(cfg, deploymentClient, repo)
	if err != nil {
		return nil, err
	}
	owner := cfg.owner

	// params
	if err := deploymentService.ValidateRepo(ctx); err != nil {
		fmt.Println(err)
		fmt.Println(fmt.Errorf("no repository found for workload %s", workload))
	}

	fmt.Printf("owner: %s\n", owner)

	//queryFrom := os.Getenv("FROM")
	//queryTo := os.Getenv("TO")
	//
	//params, err := fillParams(queryFrom, queryTo)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//deployments, err := deploymentService.ListDeploymentsInRange(ctx, params.From, params.To)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//fmt.Printf("len deploys: %d\n", len(deployments))
	//fmt.Printf("deployments response: %+v\n", deployments)

	from, err := time.Parse(time.RFC3339, "2026-02-16T01:00:00+01:00")
	to, err := time.Parse(time.RFC3339, "2026-02-16T01:20:00+01:00")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	newerDeployments, err := deploymentService.ListDeploymentsInRange(ctx, from, to)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &DeploymentResponse{Deployments: newerDeployments}, nil

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
