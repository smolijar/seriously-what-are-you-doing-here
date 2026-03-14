package main

import (
	"context"
	"fmt"
	"time"
)

type SmokeTestReport struct {
	Slack  SlackSmokeResult
	GitHub GitHubSmokeResult
}

type MonthResult struct {
	Month        string
	SlackCount   int
	GitHubCount  int
	SlackPath    string
	GitHubPath   string
	ManifestPath string
	GeneratedAt  time.Time
}

func smokeTest(ctx context.Context, cfg Config) (SmokeTestReport, error) {
	githubCollector := GitHubCollector{}
	githubReport, err := githubCollector.SmokeTest(ctx, cfg)
	if err != nil {
		return SmokeTestReport{}, err
	}
	slackCollector, err := newSlackCollector(ctx, cfg)
	if err != nil {
		return SmokeTestReport{}, err
	}
	slackReport, err := slackCollector.SmokeTest(ctx)
	if err != nil {
		return SmokeTestReport{}, err
	}
	return SmokeTestReport{Slack: slackReport, GitHub: githubReport}, nil
}

func collectMonth(ctx context.Context, cfg Config, month MonthRange) (MonthResult, error) {
	slackCollector, err := newSlackCollector(ctx, cfg)
	if err != nil {
		return MonthResult{}, err
	}
	githubCollector := GitHubCollector{}
	return collectMonthWithCollectors(ctx, cfg, month, slackCollector, githubCollector)
}

func collectMonthWithCollectors(ctx context.Context, cfg Config, month MonthRange, slackCollector *SlackCollector, githubCollector GitHubCollector) (MonthResult, error) {
	slackRecords, err := slackCollector.CollectMonth(ctx, cfg, month)
	if err != nil {
		return MonthResult{}, err
	}

	githubRecords, err := githubCollector.CollectMonth(ctx, cfg, month)
	if err != nil {
		return MonthResult{}, err
	}

	result, err := writeMonthOutput(cfg, month, slackRecords, githubRecords)
	if err != nil {
		return MonthResult{}, fmt.Errorf("write output: %w", err)
	}
	return result, nil
}
