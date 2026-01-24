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
	policySvc := service.NewPolicyService(workflows, ghClient)
	remediationSvc := service.NewRemediationService(ghClient)

	bot := orchestrator.NewGithubActionsBot(repoSvc, policySvc, remediationSvc)

	results, err := bot.Run(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("Error: %s in %s - %v\n", r.Drift.Policy.Name, r.Drift.Repository.FullName, r.Error)
		} else {
			fmt.Printf("Remediation: %s in %s - %s (%s)\n", r.Drift.Policy.Name, r.Drift.Repository.FullName, r.Action, r.PRURL)
		}
	}
}
