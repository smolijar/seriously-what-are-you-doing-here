package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type GitHubCollector struct{}

type GitHubCommitRecord struct {
	Month         string     `json:"month"`
	Repo          string     `json:"repo"`
	SHA           string     `json:"sha"`
	CommittedAt   string     `json:"committed_at"`
	AuthorLogin   string     `json:"author_login,omitempty"`
	AuthorName    string     `json:"author_name,omitempty"`
	Message       string     `json:"message"`
	HTMLURL       string     `json:"html_url"`
	IsMergeCommit bool       `json:"is_merge_commit"`
	Parents       []string   `json:"parents,omitempty"`
	SourceWindow  TimeWindow `json:"source_window"`
}

type ghCommit struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
	Author  *struct {
		Login string `json:"login"`
	} `json:"author"`
	Commit struct {
		Author struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}

func (g GitHubCollector) SmokeTest(ctx context.Context, cfg Config) (GitHubSmokeResult, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return GitHubSmokeResult{}, fmt.Errorf("gh not found in PATH: %w", err)
	}
	if _, err := runCommand(ctx, "gh", "auth", "status"); err != nil {
		return GitHubSmokeResult{}, fmt.Errorf("gh auth status failed: %w", err)
	}
	for _, repo := range cfg.Repos {
		if _, err := runCommand(ctx, "gh", "api", "repos/"+repo); err != nil {
			return GitHubSmokeResult{}, fmt.Errorf("gh cannot access repo %s: %w", repo, err)
		}
	}
	return GitHubSmokeResult{RepoCount: len(cfg.Repos)}, nil
}

func (g GitHubCollector) CollectMonth(ctx context.Context, cfg Config, month MonthRange) ([]GitHubCommitRecord, error) {
	monthStart, monthEnd := clipMonth(month, cfg)
	window := TimeWindow{From: monthStart.Format(time.RFC3339), To: monthEnd.Format(time.RFC3339)}
	var records []GitHubCommitRecord
	for _, repo := range cfg.Repos {
		commits, err := fetchRepoCommits(ctx, repo, cfg.GitHubUser, monthStart, monthEnd)
		if err != nil {
			return nil, err
		}
		for _, commit := range commits {
			record := GitHubCommitRecord{
				Month:         month.Label,
				Repo:          repo,
				SHA:           commit.SHA,
				CommittedAt:   commit.Commit.Author.Date,
				AuthorName:    commit.Commit.Author.Name,
				Message:       commit.Commit.Message,
				HTMLURL:       commit.HTMLURL,
				IsMergeCommit: len(commit.Parents) > 1,
				SourceWindow:  window,
			}
			if commit.Author != nil {
				record.AuthorLogin = commit.Author.Login
			}
			for _, parent := range commit.Parents {
				record.Parents = append(record.Parents, parent.SHA)
			}
			records = append(records, record)
		}
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Repo == records[j].Repo {
			return records[i].CommittedAt < records[j].CommittedAt
		}
		return records[i].Repo < records[j].Repo
	})
	return records, nil
}

func fetchRepoCommits(ctx context.Context, repo, author string, since, until time.Time) ([]ghCommit, error) {
	args := []string{
		"api",
		"repos/" + repo + "/commits",
		"--method", "GET",
		"--paginate",
		"--slurp",
		"-f", "author=" + author,
		"-f", "since=" + since.Format(time.RFC3339),
		"-f", "until=" + until.Format(time.RFC3339),
		"-f", "per_page=100",
	}
	output, err := runCommand(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("fetch commits for %s: %w", repo, err)
	}
	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return nil, nil
	}

	var pages [][]ghCommit
	if err := json.Unmarshal(output, &pages); err != nil {
		return nil, fmt.Errorf("decode gh commits for %s: %w", repo, err)
	}
	var commits []ghCommit
	for _, page := range pages {
		commits = append(commits, page...)
	}
	return commits, nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, message)
	}
	return output, nil
}

type GitHubSmokeResult struct {
	RepoCount int
}
