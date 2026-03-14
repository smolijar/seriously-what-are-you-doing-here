package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type MonthManifest struct {
	Month              string   `json:"month"`
	GeneratedAt        string   `json:"generated_at"`
	OutputDir          string   `json:"output_dir"`
	SlackRecordCount   int      `json:"slack_record_count"`
	GitHubCommitCount  int      `json:"github_commit_count"`
	Repos              []string `json:"repos"`
	SlackUserHandle    string   `json:"slack_user_handle"`
	GitHubUserHandle   string   `json:"github_user_handle"`
	ConfiguredTimeFrom string   `json:"configured_time_from"`
	ConfiguredTimeTo   string   `json:"configured_time_to"`
	Files              FileSet  `json:"files"`
	Warnings           []string `json:"warnings,omitempty"`
}

type FileSet struct {
	Slack    string `json:"slack"`
	GitHub   string `json:"github"`
	Manifest string `json:"manifest"`
}

func writeMonthOutput(cfg Config, month MonthRange, slackRecords []SlackConversationRecord, githubRecords []GitHubCommitRecord) (MonthResult, error) {
	monthDir := filepath.Join(cfg.OutputDir, month.Label)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return MonthResult{}, err
	}

	slackPath := filepath.Join(monthDir, "slack.jsonl")
	githubPath := filepath.Join(monthDir, "github.jsonl")
	manifestPath := filepath.Join(monthDir, "manifest.json")

	if err := writeJSONL(slackPath, slackRecords); err != nil {
		return MonthResult{}, err
	}
	if err := writeJSONL(githubPath, githubRecords); err != nil {
		return MonthResult{}, err
	}

	generatedAt := time.Now().UTC()
	manifest := MonthManifest{
		Month:              month.Label,
		GeneratedAt:        generatedAt.Format(time.RFC3339),
		OutputDir:          monthDir,
		SlackRecordCount:   len(slackRecords),
		GitHubCommitCount:  len(githubRecords),
		Repos:              append([]string(nil), cfg.Repos...),
		SlackUserHandle:    cfg.SlackUserHandle,
		GitHubUserHandle:   cfg.GitHubUser,
		ConfiguredTimeFrom: cfg.TimeFrom.Format(time.RFC3339),
		ConfiguredTimeTo:   cfg.TimeTo.Format(time.RFC3339),
		Files: FileSet{
			Slack:    slackPath,
			GitHub:   githubPath,
			Manifest: manifestPath,
		},
	}
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return MonthResult{}, err
	}

	return MonthResult{
		Month:        month.Label,
		SlackCount:   len(slackRecords),
		GitHubCount:  len(githubRecords),
		SlackPath:    slackPath,
		GitHubPath:   githubPath,
		ManifestPath: manifestPath,
		GeneratedAt:  generatedAt,
	}, nil
}

func writeJSONL[T any](path string, values []T) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return fmt.Errorf("encode %s: %w", path, err)
		}
	}
	return writer.Flush()
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func printSmokeTestReport(report SmokeTestReport) {
	fmt.Printf("Slack OK: workspace=%s user_id=%s channels=%d users=%d\n", report.Slack.WorkspaceURL, report.Slack.UserID, report.Slack.ChannelCount, report.Slack.UserCount)
	fmt.Printf("GitHub OK: repos=%d\n", report.GitHub.RepoCount)
}

func printMonthResult(result MonthResult) {
	fmt.Printf("%s: slack=%d github=%d\n", result.Month, result.SlackCount, result.GitHubCount)
	fmt.Printf("  slack: %s\n", result.SlackPath)
	fmt.Printf("  github: %s\n", result.GitHubPath)
	fmt.Printf("  manifest: %s\n", result.ManifestPath)
}
