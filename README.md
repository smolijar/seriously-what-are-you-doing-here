<div align="center">

# swaydh (Seriously, What Are You Doing Here?)

![](https://i.imgflip.com/anxrs1.gif)

</div>

Tiny CLI data aggregator for Slack + GitHub for user's footprint.

If I were to use this, I would probably do that to gather inputs I could later process to find mentionworthy items to a CV, to create a backed dataset for Gergely Orosz's inspired items. Then maybe process the results with something like described in Achievement bank. I would create the bank on monthly/quarterly bases based on the total evaluated period lenght, as it might yield better results.

## What it could maybe pull, allegedly

- Slack messages written by the target human
- Slack messages that explicitly ping them as `<@USERID>`
- Whole Slack threads when one of those messages lives in a thread
- GitHub commits for the configured GitHub handle across the configured repos

It writes stuff to:

- `OUTPUT_DIR/YYYY-MM/slack.jsonl`
- `OUTPUT_DIR/YYYY-MM/github.jsonl`
- `OUTPUT_DIR/YYYY-MM/manifest.json`

## Needs

- Go `1.25+`
- `gh` installed and logged in
- Slack token + cookie in env

## Setup speedrun

### 1. GitHub auth

```bash
gh auth login
gh auth status
```

### 2. Slack auth

Fast path:

```bash
./swaydh auth-env --workspace your-workspace
```

That opens the browser dance and spits out exports like:

```bash
export SLACK_TOKEN="xoxc-..."
export SLACK_COOKIE="xoxd-..."
```

If you insist on raw gremlin mode, get them from an authenticated Slack browser session:

```javascript
JSON.parse(localStorage.localConfig_v2).teams[document.location.pathname.match(/^\/client\/([A-Z0-9]+)/)[1]].token
```

Then grab cookie `d` and put it in `SLACK_COOKIE`.

### 3. Env file

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

`SLACK_USER_ID` is optional, but giving it saves the tool from having to go look it up.

## Build

```bash
go build -o swaydh .
```

## Commands

If `.env` exists in the repo root, commands load it automatically. Use `--env-file` if you want to be fancy.

### Smoke test

Checks whether this whole scheme could theoretically work before you waste your afternoon.

```bash
./swaydh smoke-test
```

### Preview one month

```bash
./swaydh preview --month 2026-02
```

Re-run and overwrite that month:

```bash
./swaydh preview --month 2026-02 --force
```

### Run the whole range

```bash
./swaydh run
```

Overwrite already-generated months:

```bash
./swaydh run --force
```

Use a different env file:

```bash
./swaydh run --env-file /path/to/file.env
```

## Basically the workflow

```bash
go build -o swaydh .
./swaydh auth-env --workspace your-workspace
./swaydh smoke-test
./swaydh preview --month 2026-02
./swaydh run
```

## Achievement bank

After the monthly output exists, you can feed it into the achievement-bank thing:

- `docs/achievement-bank-manual.md`
- `docs/achievement-bank-prompt.md`

That step is for generating a reusable achievement bank, not a resume, not a monthly diary, not LinkedIn fanfic.

## Output

- `slack.jsonl`: one JSON object per matched conversation unit
- `github.jsonl`: one JSON object per commit
- `manifest.json`: metadata about that month's run

## Note

- Slack mentions means real Slack mentions, not plain text name-dropping
- Results only cover conversations visible to the authenticated Slack account
- Free Slack may refuse to cough up stuff older than 90 days
- `TIME_TO` is exclusive
- `preview --month YYYY-MM` has to fall inside `TIME_FROM` and `TIME_TO`
- If your org hates automated Slack access, maybe do not become the main character
