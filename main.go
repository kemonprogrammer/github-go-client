package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kemonprogrammer/github-go-client/config"
	"github.com/kemonprogrammer/github-go-client/external_deployments/types"
	"github.com/kemonprogrammer/github-go-client/handler"
)

type Response struct {
	Deployments []*types.Deployment `json:"deployments"`
	Size        int                 `json:"total"`
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

	workload := os.Getenv("WORKLOAD")

	wg := sync.WaitGroup{}
	var newerDeployments []*types.Deployment
	wg.Add(1)
	go func() {
		defer wg.Done()
		resp, err := handler.HttpHandler(context.Background(), cfg, workload)
		if err != nil {
			fmt.Println(err)
			return
		}
		newerDeployments = resp.Deployments
		fmt.Printf("len newer deploys: %d\n", len(newerDeployments))

		fmt.Printf("newer deployments response: %+v\n", newerDeployments)
	}()

	//// -- 2nd call
	//wg.Add(1)
	//go func() {
	//	defer wg.Done()
	//	resp, err := handler.HttpHandler(context.Background(), cfg, workload)
	//	if err != nil {
	//		fmt.Println(err)
	//		return
	//	}
	//	newerDeployments := resp.Deployments
	//	fmt.Printf("2nd time to test cache: deployments response: %+v\n", newerDeployments)
	//}()

	wg.Wait()

	// 1. Initialize your data
	res := Response{
		Deployments: newerDeployments,
		Size:        len(newerDeployments),
	}

	// 2. Marshal to JSON (returns []byte)
	jsonData, err := json.Marshal(res)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	// 3. Convert []byte to string
	jsonString := string(jsonData)

	fmt.Printf("%s", jsonString)

}

func SetupConfig() *config.Config {
	// setup github
	return &config.Config{
		Owner:    os.Getenv("OWNER"),
		Env:      os.Getenv("ENVIRONMENT"),
		Token:    os.Getenv("GITHUB_PAT"),
		Enabled:  true,
		Provider: "github",
	}
}
