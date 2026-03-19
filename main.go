package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "auth-env":
		return runAuthEnv(ctx, args[1:])
	case "smoke-test":
		return runSmokeTest(ctx, args[1:])
	case "preview":
		return runPreview(ctx, args[1:])
	case "run":
		return runFull(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSmokeTest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("smoke-test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	envFile := fs.String("env-file", "", "path to env file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*envFile)
	if err != nil {
		return err
	}

	report, err := smokeTest(ctx, cfg)
	if err != nil {
		return err
	}

	printSmokeTestReport(report)
	return nil
}

func runAuthEnv(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("auth-env", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	workspace := fs.String("workspace", "", "Slack workspace name or URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*workspace) == "" {
		return errors.New("auth-env requires --workspace <workspace-name-or-url>")
	}

	result, err := getSlackAuthEnv(ctx, *workspace)
	if err != nil {
		return err
	}

	printSlackAuthEnv(result)
	return nil
}

func runPreview(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	envFile := fs.String("env-file", "", "path to env file")
	monthValue := fs.String("month", "", "month to preview in YYYY-MM")
	force := fs.Bool("force", false, "overwrite existing month output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *monthValue == "" {
		return errors.New("preview requires --month YYYY-MM")
	}

	cfg, err := loadConfig(*envFile)
	if err != nil {
		return err
	}

	month, err := parseMonth(*monthValue)
	if err != nil {
		return err
	}
	if !month.End.After(cfg.TimeFrom) || !month.Start.Before(cfg.TimeTo) {
		return fmt.Errorf("month %s is outside configured range %s to %s", month.Label, cfg.TimeFrom.Format(time.DateOnly), cfg.TimeTo.Format(time.DateOnly))
	}
	if exists, path, err := monthOutputExists(cfg, month); err != nil {
		return err
	} else if exists && !*force {
		logf("[%s] skipping existing output at %s (use --force to overwrite)", month.Label, path)
		return nil
	}

	result, err := collectMonth(ctx, cfg, month)
	if err != nil {
		return err
	}

	printMonthResult(result)
	return nil
}

func runFull(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	envFile := fs.String("env-file", "", "path to env file")
	force := fs.Bool("force", false, "overwrite existing month output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadConfig(*envFile)
	if err != nil {
		return err
	}

	months := monthsInRange(cfg.TimeFrom, cfg.TimeTo)
	if len(months) == 0 {
		return errors.New("no months in requested range")
	}

	slackCollector, err := newSlackCollector(ctx, cfg)
	if err != nil {
		return err
	}
	githubCollector := GitHubCollector{}

	for _, month := range months {
		if exists, path, err := monthOutputExists(cfg, month); err != nil {
			return err
		} else if exists && !*force {
			logf("[%s] skipping existing output at %s (use --force to overwrite)", month.Label, path)
			continue
		}
		result, err := collectMonthWithCollectors(ctx, cfg, month, slackCollector, githubCollector)
		if err != nil {
			return fmt.Errorf("collect %s: %w", month.Label, err)
		}
		printMonthResult(result)
	}

	return nil
}

func printUsage() {
	text := strings.TrimSpace(`swaydh

Seriously What Are you Doing Here

Usage:
  swaydh auth-env --workspace my-workspace
  swaydh smoke-test [--env-file .env]
  swaydh preview --month YYYY-MM [--env-file .env] [--force]
  swaydh run [--env-file .env] [--force]

The tool reads all configuration from environment variables or an optional env file.
See .env.example and README.md for setup instructions.
`)
	fmt.Println(text)
}
