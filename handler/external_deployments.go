package handler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/kemonprogrammer/github-go-client/config"
	"github.com/kemonprogrammer/github-go-client/external_deployments"
	"github.com/kemonprogrammer/github-go-client/external_deployments/model"
	"github.com/kemonprogrammer/github-go-client/models"
)

func HttpHandler(ctx context.Context, conf *config.Config, workload string) (*DeploymentResponse, error) {
	repo := extractRepoName(workload)

	deploymentClient, err := external_deployments.NewDeploymentClient(conf)
	if err != nil {
		return nil, err
	}
	deploymentService, err := external_deployments.NewDeploymentService(conf, deploymentClient)
	if err != nil {
		return nil, err
	}

	owner := conf.Owner

	// params
	if err := deploymentService.SetRepo(ctx, repo); err != nil {
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

	from, err := time.Parse(time.RFC3339, "2026-03-18T02:00:00+01:00")
	to, err := time.Parse(time.RFC3339, "2026-03-18T03:00:00+01:00")
	//from = time.Now().Add(-10 * time.Minute)
	//to = time.Now()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	q := models.DeploymentsQuery{
		From:      from,
		To:        to,
		Cluster:   "",
		Namespace: "",
		Workload:  "",
	}

	deployments, err := deploymentService.ListDeploymentsInRange(ctx, q)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &DeploymentResponse{Deployments: deployments}, nil

}

type DeploymentResponse struct {
	Deployments []*model.Deployment `json:"deployments"`
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
