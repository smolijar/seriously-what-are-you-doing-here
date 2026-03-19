package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	SlackToken      string
	SlackCookie     string
	SlackUserHandle string
	SlackUserID     string
	GitHubUser      string
	Repos           []string
	TimeFrom        time.Time
	TimeTo          time.Time
	OutputDir       string
}

func loadConfig(envFile string) (Config, error) {
	if err := loadEnvFile(envFile); err != nil {
		return Config{}, err
	}

	var cfg Config
	cfg.SlackToken = strings.TrimSpace(os.Getenv("SLACK_TOKEN"))
	cfg.SlackCookie = strings.TrimSpace(os.Getenv("SLACK_COOKIE"))
	cfg.SlackUserHandle = normalizeHandle(os.Getenv("SLACK_USER_HANDLE"))
	cfg.SlackUserID = strings.TrimSpace(os.Getenv("SLACK_USER_ID"))
	cfg.GitHubUser = normalizeHandle(os.Getenv("GITHUB_USER_HANDLE"))

	repos, err := parseCSVList(os.Getenv("REPOS"))
	if err != nil {
		return Config{}, fmt.Errorf("REPOS: %w", err)
	}
	cfg.Repos = repos

	from, err := parseDateEnv("TIME_FROM")
	if err != nil {
		return Config{}, err
	}
	to, err := parseDateEnv("TIME_TO")
	if err != nil {
		return Config{}, err
	}
	if !from.Before(to) {
		return Config{}, errors.New("TIME_FROM must be before TIME_TO")
	}
	cfg.TimeFrom = from.UTC()
	cfg.TimeTo = to.UTC()

	outputDir := strings.TrimSpace(os.Getenv("OUTPUT_DIR"))
	if outputDir == "" {
		outputDir = "out"
	}
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve OUTPUT_DIR: %w", err)
	}
	cfg.OutputDir = absOutput

	missing := missingFields(map[string]string{
		"SLACK_TOKEN":        cfg.SlackToken,
		"SLACK_COOKIE":       cfg.SlackCookie,
		"GITHUB_USER_HANDLE": cfg.GitHubUser,
		"TIME_FROM":          os.Getenv("TIME_FROM"),
		"TIME_TO":            os.Getenv("TIME_TO"),
		"REPOS":              os.Getenv("REPOS"),
	})
	if cfg.SlackUserHandle == "" && cfg.SlackUserID == "" {
		missing = append(missing, "SLACK_USER_HANDLE or SLACK_USER_ID")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func loadEnvFile(envFile string) error {
	if envFile != "" {
		if err := godotenv.Overload(envFile); err != nil {
			return fmt.Errorf("load env file %q: %w", envFile, err)
		}
		return nil
	}

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return fmt.Errorf("load default .env: %w", err)
		}
	}
	return nil
}

func parseCSVList(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if !strings.Contains(value, "/") {
			return nil, fmt.Errorf("invalid repo %q, expected owner/repo", value)
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil, errors.New("must include at least one owner/repo value")
	}
	return values, nil
}

func parseDateEnv(key string) (time.Time, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Time{}, fmt.Errorf("%s is required", key)
	}
	parsed, err := time.ParseInLocation(time.DateOnly, raw, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must use YYYY-MM-DD: %w", key, err)
	}
	return parsed.UTC(), nil
}

func missingFields(values map[string]string) []string {
	missing := make([]string, 0)
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func normalizeHandle(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "@")
}
