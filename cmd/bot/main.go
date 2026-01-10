package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/tracker-tv/github-policy-bots/internal/config"
	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/internal/policy"
)

//go:embed policies/*.json
var embeddedPolicies embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	fmt.Printf("Config: %+v\n", cfg)

	data, err := embeddedPolicies.ReadFile("policies/github-actions.json")
	if err != nil {
		log.Fatalln(err)
	}

	workflows, err := policy.FromJSON(data)
	if err != nil {
		log.Fatalln(err)
	}

	client := github.New(cfg.GithubPAT)
	repos, err := client.ListAllRepos(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	for _, repo := range repos {
		fmt.Println(repo.GetName())
		fmt.Printf("%+v\n", repo)
	}

	fmt.Printf("%+v\n", workflows)
}
