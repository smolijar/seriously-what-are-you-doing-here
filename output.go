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

type MonthWriter struct {
	cfg          Config
	month        MonthRange
	monthDir     string
	slackPath    string
	githubPath   string
	manifestPath string
	generatedAt  time.Time

	slackFile   *os.File
	githubFile  *os.File
	slackBuf    *bufio.Writer
	githubBuf   *bufio.Writer
	slackEnc    *json.Encoder
	githubEnc   *json.Encoder
	slackCount  int
	githubCount int
}

func newMonthWriter(cfg Config, month MonthRange) (*MonthWriter, error) {
	monthDir := filepath.Join(cfg.OutputDir, month.Label)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return nil, err
	}

	w := &MonthWriter{
		cfg:          cfg,
		month:        month,
		monthDir:     monthDir,
		slackPath:    monthOutputPaths(cfg, month).Slack,
		githubPath:   monthOutputPaths(cfg, month).GitHub,
		manifestPath: monthOutputPaths(cfg, month).Manifest,
	}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func monthOutputPaths(cfg Config, month MonthRange) FileSet {
	monthDir := filepath.Join(cfg.OutputDir, month.Label)
	return FileSet{
		Slack:    filepath.Join(monthDir, "slack.jsonl"),
		GitHub:   filepath.Join(monthDir, "github.jsonl"),
		Manifest: filepath.Join(monthDir, "manifest.json"),
	}
}

func monthOutputExists(cfg Config, month MonthRange) (bool, string, error) {
	paths := monthOutputPaths(cfg, month)
	for _, path := range []string{paths.Manifest, paths.Slack, paths.GitHub} {
		if _, err := os.Stat(path); err == nil {
			return true, path, nil
		} else if !os.IsNotExist(err) {
			return false, "", err
		}
	}
	return false, "", nil
}

func (w *MonthWriter) open() error {
	var err error
	w.slackFile, err = os.Create(w.slackPath)
	if err != nil {
		return err
	}
	w.githubFile, err = os.Create(w.githubPath)
	if err != nil {
		_ = w.slackFile.Close()
		return err
	}
	w.slackBuf = bufio.NewWriter(w.slackFile)
	w.githubBuf = bufio.NewWriter(w.githubFile)
	w.slackEnc = json.NewEncoder(w.slackBuf)
	w.githubEnc = json.NewEncoder(w.githubBuf)
	w.slackEnc.SetEscapeHTML(false)
	w.githubEnc.SetEscapeHTML(false)
	return nil
}

func (w *MonthWriter) AppendSlack(record SlackConversationRecord) error {
	if err := w.slackEnc.Encode(record); err != nil {
		return fmt.Errorf("encode %s: %w", w.slackPath, err)
	}
	w.slackCount++
	return w.slackBuf.Flush()
}

func (w *MonthWriter) AppendGitHub(record GitHubCommitRecord) error {
	if err := w.githubEnc.Encode(record); err != nil {
		return fmt.Errorf("encode %s: %w", w.githubPath, err)
	}
	w.githubCount++
	return w.githubBuf.Flush()
}

func (w *MonthWriter) Close() error {
	var firstErr error
	if w.slackBuf != nil {
		if err := w.slackBuf.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.githubBuf != nil {
		if err := w.githubBuf.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.slackFile != nil {
		if err := w.slackFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if w.githubFile != nil {
		if err := w.githubFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (w *MonthWriter) Finalize() (MonthResult, error) {
	w.generatedAt = time.Now().UTC()
	manifest := MonthManifest{
		Month:              w.month.Label,
		GeneratedAt:        w.generatedAt.Format(time.RFC3339),
		OutputDir:          w.monthDir,
		SlackRecordCount:   w.slackCount,
		GitHubCommitCount:  w.githubCount,
		Repos:              append([]string(nil), w.cfg.Repos...),
		SlackUserHandle:    w.cfg.SlackUserHandle,
		GitHubUserHandle:   w.cfg.GitHubUser,
		ConfiguredTimeFrom: w.cfg.TimeFrom.Format(time.RFC3339),
		ConfiguredTimeTo:   w.cfg.TimeTo.Format(time.RFC3339),
		Files: FileSet{
			Slack:    w.slackPath,
			GitHub:   w.githubPath,
			Manifest: w.manifestPath,
		},
	}
	if err := writeJSONFile(w.manifestPath, manifest); err != nil {
		return MonthResult{}, err
	}
	return MonthResult{
		Month:        w.month.Label,
		SlackCount:   w.slackCount,
		GitHubCount:  w.githubCount,
		SlackPath:    w.slackPath,
		GitHubPath:   w.githubPath,
		ManifestPath: w.manifestPath,
		GeneratedAt:  w.generatedAt,
	}, nil
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
