package handler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/kemonprogrammer/github-go-client/config"
	"github.com/kemonprogrammer/github-go-client/external_deployments"
	"github.com/kemonprogrammer/github-go-client/external_deployments/types"
)

func HttpHandler(ctx context.Context, cfg *config.Config, workload string) (*DeploymentResponse, error) {
	repo := extractRepoName(workload)
	deploymentService, err := external_deployments.NewDeploymentService(cfg, repo)
	if err != nil {
		return nil, err
	}
	owner := cfg.Owner

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

	deployments, err := deploymentService.ListDeploymentsInRange(ctx, from, to)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &DeploymentResponse{Deployments: deployments}, nil

}

type DeploymentResponse struct {
	Deployments []*types.Deployment `json:"deployments"`
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
