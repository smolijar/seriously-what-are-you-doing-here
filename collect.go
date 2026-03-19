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
	writer, err := newMonthWriter(cfg, month)
	if err != nil {
		return MonthResult{}, err
	}
	defer writer.Close()

	progress := NewProgressReporter(month, cfg)
	progress.StartMonth()

	_, err = slackCollector.StreamMonth(ctx, cfg, month, progress, writer.AppendSlack)
	if err != nil {
		return MonthResult{}, err
	}

	_, err = githubCollector.StreamMonth(ctx, cfg, month, progress, writer.AppendGitHub)
	if err != nil {
		return MonthResult{}, err
	}

	result, err := writer.Finalize()
	if err != nil {
		return MonthResult{}, fmt.Errorf("write output: %w", err)
	}
	logf("[%s] done %s", month.Label, progress.Summary())
	return result, nil
}
