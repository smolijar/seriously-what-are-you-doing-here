# swaydh

swaydh stands for Seriously What Are you Doing Here.

This tool collects month-by-month Slack and GitHub activity for a specific person.

It reads all configuration from environment variables, can smoke-test access to Slack and GitHub, can preview a single month, and can run across the full configured time range.

## What it collects

- Slack messages authored by the target user
- Slack messages that explicitly mention the target user as `<@USERID>`
- Whole Slack threads when a matched message belongs to a thread
- GitHub commits for the target GitHub handle across the configured repo list

Outputs are written per month:

- `OUTPUT_DIR/YYYY-MM/slack.jsonl`
- `OUTPUT_DIR/YYYY-MM/github.jsonl`
- `OUTPUT_DIR/YYYY-MM/manifest.json`

## Requirements

- Go 1.25+
- `gh` installed and authenticated
- Slack web token and cookie values available in env

## Setup

### 1. Authenticate GitHub CLI

Install `gh`, then log in:

```bash
gh auth login
gh auth status
```

### 2. Get Slack credentials

The tool uses the `slackdump` Go library with runtime env credentials.

Get the values from an authenticated Slack browser session:

1. Open your Slack workspace in a browser.
2. Open browser developer tools.
3. In the console, run:

```javascript
JSON.parse(localStorage.localConfig_v2).teams[document.location.pathname.match(/^\/client\/([A-Z0-9]+)/)[1]].token
```

4. Copy the returned `xoxc-...` token into `SLACK_TOKEN`.
5. In browser storage/cookies, find the Slack cookie named `d`.
6. Copy its value into `SLACK_COOKIE`.

Slack workspaces may trigger security alerts for automated access. Make sure your usage complies with your organization's policies.

### 3. Create your env file

Copy the example file and fill in your values:

```bash
cp .env.example .env
```

Example:

```dotenv
SLACK_TOKEN=xoxc-...
SLACK_COOKIE=xoxd-...
SLACK_USER_HANDLE=alice
GITHUB_USER_HANDLE=alice
REPOS=org/repo-a,org/repo-b
TIME_FROM=2026-01-01
TIME_TO=2026-04-01
OUTPUT_DIR=out
```

You can also export everything directly in your shell instead of using `.env`.

## Build

```bash
go build -o swaydh .
```

## Usage

If `.env` exists in the project root, it is loaded automatically.

Smoke test access:

```bash
./swaydh smoke-test
```

Run a single month preview:

```bash
./swaydh preview --month 2026-02
```

Run the full configured range:

```bash
./swaydh run
```

Use a different env file:

```bash
./swaydh run --env-file /path/to/file.env
```

## Output format

`slack.jsonl` contains one JSON object per matched conversation unit:

- single message for non-thread matches
- full thread for thread matches

Each record includes the month, channel metadata, match type, matched message timestamp, source window, and compact message payloads.

`github.jsonl` contains one JSON object per commit with repo, SHA, author details, commit message, URL, parent SHAs, and source window.

`manifest.json` contains run metadata and output file paths for that month.

## Notes

- Slack mentions are explicit Slack mentions only, not plain-text name references.
- Slack results only cover channels, private channels, DMs, and group DMs visible to the authenticated Slack account.
- Free Slack workspaces may not return data older than 90 days.
- `TIME_TO` is treated as the exclusive upper bound.
