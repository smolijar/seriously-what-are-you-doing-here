# Achievement Bank Generation Manual

## Goal

Turn collected month-by-month Slack and GitHub activity into a reusable achievement bank for one person's role.

The achievement bank is not a monthly summary.
It is not a changelog.
It is not a list of responsibilities.

It is a compact set of high-signal, reusable professional achievements supported by evidence from the collected data.

The output should follow the principles from Gergely Orosz's resume guidance:

- prioritize relevance and clarity
- prefer outcomes over duties
- use specifics over vague claims
- include technologies only where they add signal
- quantify impact when evidence supports it

## Inputs

Read all available monthly output folders under `out/YYYY-MM/`.

Each month may contain:

- `manifest.json`
- `github.jsonl`
- `slack.jsonl`

Use all available months unless explicitly told otherwise.

### File meanings

- `github.jsonl`
  - one record per commit
  - useful for identifying shipped work, technical themes, fixes, migrations, performance work, and implementation details

- `slack.jsonl`
  - one record per matched message or matched thread
  - useful for identifying context, cross-team coordination, incident handling, decision-making, metrics, and business impact

- `manifest.json`
  - useful for date range, repo scope, and collection completeness
  - not a source of achievement content

## Required mindset

You are not trying to summarize every activity item.
You are trying to identify a small number of important achievements that would help a recruiter or hiring manager understand this person's impact.

Prefer fewer strong achievements over many weak ones.

## Output definition

Produce a reusable achievement bank as structured Markdown.

The bank should contain:

1. A brief overview of the strongest recurring themes
2. 6-15 achievement entries total
3. Each entry must include:
   - title
   - time span
   - summary
   - impact
   - supporting evidence
   - confidence
   - reusable resume bullet
   - tags

## Noise reduction rules

### Always de-emphasize or ignore

- trivial status updates
- conversational filler
- isolated links with no context
- jokes, acknowledgements, reactions, or casual chat
- messages that only say work is in progress without clarifying why it matters
- cosmetic-only work unless it clearly mattered to users or a launch
- merge commits when the underlying non-merge commits already describe the work
- repeated commits or messages describing the same change
- one-off admin tweaks unless they indicate ownership of a meaningful workflow or business process

### Treat as weak evidence unless supported by more context

- DMs
- short Slack messages without outcome or scope
- commit titles that are too terse or ambiguous
- messages describing intent without delivery
- claims of impact without numbers or corroboration

### Treat as strong evidence

- fixes to production-impacting bugs
- performance improvements with before/after evidence
- features tied to adoption, conversion, reliability, scale, or internal efficiency
- recurring ownership across related commits or threads
- launch, integration, or migration work
- cross-functional coordination that unblocked delivery
- metrics, counts, percentages, time reductions, or volume numbers
- work spanning multiple months that indicates initiative ownership

## Evidence extraction rules

For each useful record, extract candidate evidence in this normalized shape:

- date
- source (`github` or `slack`)
- month
- theme
- initiative candidate
- action type
- short evidence summary
- impact signal
- tech or context
- raw support reference

### Action type examples

- feature_delivery
- bug_fix
- performance_improvement
- reliability_improvement
- migration
- integration
- analytics_or_metrics
- cross_team_coordination
- incident_response
- process_improvement
- product_iteration

### Theme examples

- onboarding
- invites
- payments
- landlord integrations
- insurance
- dashboards
- messaging
- data quality
- query performance
- admin tooling

## Aggregation rules

Group evidence across months into broader initiative candidates.

Do not keep achievements month-scoped unless the work truly happened only once and was still meaningful.

Merge records when they refer to:

- the same project or feature area
- the same bug or problem over time
- the same system or component being improved over several commits
- the same rollout, launch, landlord integration, or optimization effort

When grouping, prefer initiative-level framing over task-level framing.

### Good aggregation

- several commits improving invite query speed plus a Slack comment about dashboard performance
- bug fixes plus data corrections in the same insurance workflow
- multiple commits and threads related to a new landlord integration

### Bad aggregation

- combining unrelated work just because it happened in the same month
- turning every commit into its own achievement
- splitting one multi-week effort into many tiny bullets

## Achievement selection rules

Select achievements using these priorities:

1. clear business or user impact
2. clear ownership or major contribution
3. specificity of evidence
4. recurrence across time
5. relevance to common software engineering hiring signals
6. presence of quantitative indicators

Do not include low-signal entries just to hit a quota.

## Writing rules for achievement entries

Each achievement must be general and reusable.
Do not write it as if tailored to one specific job description.
Do not overfit to internal jargon if a clearer external phrasing exists.

### Summary

Write 2-4 sentences describing:

- the problem or opportunity
- what was done
- why it mattered

### Impact

State:

- measurable results if supported by evidence
- otherwise likely operational or product impact, clearly marked as inferred

Use labels:

- `explicit` for directly supported impact
- `inferred` for reasonable but not directly measured impact

### Supporting evidence

List 2-6 concrete references:

- month plus repo plus commit summary
- month plus Slack channel or thread summary
- any metrics or before-and-after statements found in source text

### Confidence

Use one of:

- high
- medium
- low

High confidence requires multiple supporting signals or explicit impact.
Low confidence means the evidence is suggestive but incomplete.

### Reusable resume bullet

Write exactly one polished bullet per achievement.
It should:

- start with a strong action verb
- focus on outcome and scope, not responsibility
- include numbers if supported
- mention technologies only when they add value
- stay concise
- avoid hedging language

Good pattern examples:

- Improved X by doing Y, resulting in Z.
- Built X to enable Y for Z users, customers, or workflows.
- Fixed X in Y system, preventing Z and improving A.

Avoid:

- Responsible for...
- Worked on...
- Helped with...
- Did various...
- Participated in...

## Handling uncertainty

Never invent facts.
Never invent scale.
Never invent team size, customer count, revenue, or percentages.

If impact is not explicit:

- keep the achievement if the work is clearly meaningful
- mark impact as inferred
- write the resume bullet conservatively

If evidence is too weak:

- drop it

## Output format

Return Markdown in exactly this structure:

```md
# Achievement Bank

## Overview
- 3-6 bullets describing the strongest recurring themes

## Achievements

### 1. <Title>
- Time span: <month range>
- Tags: <tag1>, <tag2>, <tag3>
- Confidence: <high|medium|low>

Summary:
<2-4 sentences>

Impact:
- explicit: <supported impact statements>
- inferred: <reasonable inferred impact statements, if any>

Supporting evidence:
- <evidence item 1>
- <evidence item 2>
- <evidence item 3>

Reusable resume bullet:
- <single polished bullet>

### 2. <Title>
...

## Excluded / Low-Signal Themes
- <theme or type>: <why excluded>

## Gaps / Follow-Up Questions
- <missing metric or business context that would strengthen a bullet>
```

## Final quality bar

Before finalizing, check:

- Are these achievements initiative-level, not task-level?
- Would a hiring manager care?
- Are the bullets specific?
- Are the strongest entries supported by evidence?
- Did you avoid repeating the same story in multiple entries?
- Did you avoid internal-noise Slack chatter?
- Did you avoid turning monthly collection structure into monthly output structure?
