package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rusq/slackdump/v4/auth"
)

type SlackAuthEnv struct {
	Workspace   string
	SlackToken  string
	SlackCookie string
}

func getSlackAuthEnv(ctx context.Context, workspace string) (SlackAuthEnv, error) {
	provider, err := auth.NewRODAuth(ctx, auth.BrowserWithWorkspace(workspace))
	if err != nil {
		return SlackAuthEnv{}, fmt.Errorf("automatic Slack login failed: %w", err)
	}

	dCookie, err := cookieValue(provider.Cookies(), "d")
	if err != nil {
		return SlackAuthEnv{}, err
	}

	return SlackAuthEnv{
		Workspace:   strings.TrimSpace(workspace),
		SlackToken:  provider.SlackToken(),
		SlackCookie: dCookie,
	}, nil
}

func cookieValue(cookies []*http.Cookie, name string) (string, error) {
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name != name {
			continue
		}
		if strings.TrimSpace(cookie.Value) == "" {
			break
		}
		return cookie.Value, nil
	}
	return "", fmt.Errorf("required Slack cookie %q not found in browser auth result", name)
}

func printSlackAuthEnv(env SlackAuthEnv) {
	fmt.Printf("# swaydh Slack auth env for %s\n", env.Workspace)
	fmt.Printf("export SLACK_TOKEN=%q\n", env.SlackToken)
	fmt.Printf("export SLACK_COOKIE=%q\n", env.SlackCookie)
}
