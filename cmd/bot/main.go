package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/tracker-tv/github-policy-bots/internal/config"
	"github.com/tracker-tv/github-policy-bots/internal/github"
	"github.com/tracker-tv/github-policy-bots/internal/orchestrator"
	"github.com/tracker-tv/github-policy-bots/internal/policy"
	"github.com/tracker-tv/github-policy-bots/internal/service"
)

//go:embed policies/*.json
var embeddedPolicies embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	data, err := embeddedPolicies.ReadFile("policies/github-actions.json")
	if err != nil {
		log.Fatalln(err)
	}

	workflows, err := policy.FromJSON(data)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Policies: %+v\n", workflows)

	ghClient := github.New(cfg.GithubPAT, "tracker-tv")

	repoSvc := service.NewRepositoriesService(ghClient)
	workflowSvc := service.NewWorkflowService(ghClient)

	bot := orchestrator.NewGithubActionsBot(
		repoSvc,
		workflowSvc,
	)

	if err := bot.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
