# swaydh

swaydh stands for Seriously What Are you Doing Here.

This tool collects month-by-month Slack and GitHub activity for a specific person.

Slack collection uses a search-first approach: it searches for authored messages and explicit mentions in the requested month, then fetches exact matched messages and full threads only for those hits.

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

The JSONL files are appended during processing, so partial results are preserved while a month is still running.

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

You can have `swaydh` open a browser and print the env exports for you:

```bash
./swaydh auth-env --workspace your-workspace
```

That follows the same browser-based login style described by `slackdump`: complete the login flow in the browser, then copy the printed `export` lines into your shell or `.env` file.

Expected output looks like:

```bash
# swaydh Slack auth env for your-workspace
export SLACK_TOKEN="xoxc-..."
export SLACK_COOKIE="xoxd-..."
```

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
SLACK_USER_ID=U0123456789
GITHUB_USER_HANDLE=alice
REPOS=org/repo-a,org/repo-b
TIME_FROM=2026-01-01
TIME_TO=2026-04-01
OUTPUT_DIR=out
```

`SLACK_USER_ID` is optional but recommended. If omitted, swaydh resolves it from `SLACK_USER_HANDLE` before running searches.

You can also export everything directly in your shell instead of using `.env`.

## Build

```bash
go build -o swaydh .
```

## User manual

If `.env` exists in the project root, every command loads it automatically. You can override that with `--env-file /path/to/file.env`.

### 1. Set env

Create `.env` from the example and fill in your Slack, GitHub, repo, and date range values:

```bash
cp .env.example .env
```

At minimum you need:

- `SLACK_TOKEN`
- `SLACK_COOKIE`
- `SLACK_USER_HANDLE` or `SLACK_USER_ID`
- `GITHUB_USER_HANDLE`
- `REPOS`
- `TIME_FROM`
- `TIME_TO`

### 2. Run Slack login helper

Use the helper when you need fresh Slack auth values:

```bash
./swaydh auth-env --workspace your-workspace
```

What to expect:

- a browser-based Slack login flow opens
- after login, `swaydh` prints `export` lines for `SLACK_TOKEN` and `SLACK_COOKIE`
- copy those values into your shell session or `.env`

### 3. Smoke test access

Before collecting data, verify both integrations:

```bash
./swaydh smoke-test
```

Expected output looks like:

```text
Slack OK: workspace=https://your-workspace.slack.com user_id=U0123456789 channels=123 users=456
GitHub OK: repos=2
```

This confirms:

- `gh` is installed and authenticated
- each configured repo is reachable via GitHub CLI
- the Slack token/cookie work
- the target Slack user resolves correctly
- the authenticated Slack account can see conversations

### 4. Preview one month

Run one month first before doing a full range:

```bash
./swaydh preview --month 2026-02
```

What to expect:

- progress logs while Slack and GitHub collection run
- a final summary like:

```text
2026-02: slack=14 github=6
  slack: /absolute/path/to/out/2026-02/slack.jsonl
  github: /absolute/path/to/out/2026-02/github.jsonl
  manifest: /absolute/path/to/out/2026-02/manifest.json
```

If output for that month already exists, `swaydh` skips it unless you force a rerun.

### 5. Re-run a preview month with force

Use `--force` to overwrite an existing month directory's output files:

```bash
./swaydh preview --month 2026-02 --force
```

This is useful when you changed env values, widened the time range, or want to regenerate a month after a partial run.

### 6. Run the full configured range

Once preview looks right, run the full range:

```bash
./swaydh run
```

To overwrite already-generated months:

```bash
./swaydh run --force
```

To use a non-default env file:

```bash
./swaydh run --env-file /path/to/file.env
```

What to expect:

- one summary block per month in the configured range
- months with existing output are skipped unless `--force` is set
- output files are written under `OUTPUT_DIR/YYYY-MM/`

### 7. Recommended smoke-to-run workflow

For a new setup or credential refresh, use this sequence:

```bash
go build -o swaydh .
./swaydh auth-env --workspace your-workspace
./swaydh smoke-test
./swaydh preview --month 2026-02
./swaydh preview --month 2026-02 --force
./swaydh run
```

The second preview command is only needed when you intentionally want to regenerate the same month.

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
- `preview --month YYYY-MM` must fall within the configured `TIME_FROM` / `TIME_TO` window.
