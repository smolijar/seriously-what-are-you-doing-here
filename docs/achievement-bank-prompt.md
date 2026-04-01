# Achievement Bank Prompt

Use this prompt after collection output has been generated under `out/YYYY-MM/`.

## Prompt

Read `docs/achievement-bank-manual.md` and follow it exactly.

Generate a reusable achievement bank from all collected data under `out/`.

Inputs:

- all available `out/YYYY-MM/github.jsonl`
- all available `out/YYYY-MM/slack.jsonl`
- all available `out/YYYY-MM/manifest.json`

Requirements:

- produce Markdown only
- follow the exact output structure from the manual
- prefer fewer strong achievements over many weak ones
- merge related work across months into initiative-level achievements
- explicitly separate supported impact from inferred impact
- include supporting evidence for every achievement
- exclude low-signal noise rather than summarizing everything
