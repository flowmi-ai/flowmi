# Output Templates

Choose the template that matches the triage output goal. Adapt sections as needed — these are frameworks, not rigid forms.

If the topic is time-sensitive, include a short "As of [date]" line near the top of any template.

## Brief Answer / Explainer

```markdown
# [Topic]

## TL;DR
[2-3 sentence answer]

## Key Findings
[Organized by theme. Cite inline with [Source](url).]

## Sources
[Numbered list with title and URL]
```

## Comparison

```markdown
# [Topic A] vs [Topic B] (vs [Topic C]...)

## TL;DR
[Which is better for what, in 2-3 sentences]

## Comparison Criteria
[List the dimensions being compared and why they matter]

## Side-by-Side

| Criteria | Option A | Option B | Option C |
|----------|----------|----------|----------|
| ...      | ...      | ...      | ...      |

## Trade-offs
[What you gain and lose with each option. Not everything has a winner.]

## Evidence Quality Notes
[Call out where one option has much better documentation or source coverage than another. If evidence is asymmetric, say so — a gap in coverage is not the same as a gap in capability.]

## Sources
[Numbered list]
```

## Trend Report

```markdown
# [Topic]: Trend Analysis

*As of [date]*

## TL;DR
[Where things stand and where they're heading]

## Background
[Established context — what has been true for a while]

## Timeline of Key Developments
- **[Date]**: [Event] ([Source](url))
- **[Date]**: [Event] ([Source](url))
- ...

## Current State
[What is happening now]

## Outlook
[What credible sources expect next. Distinguish predictions from facts.]

## Sources
[Numbered list]
```

## Decision Memo

```markdown
# [Decision Question]

## TL;DR
[Recommended direction and key reason, in 2-3 sentences]

## Options Considered

### Option 1: [Name]
- **Pros**: ...
- **Cons**: ...
- **Risk**: ...

### Option 2: [Name]
- **Pros**: ...
- **Cons**: ...
- **Risk**: ...

## Context Dependencies
[Which recommendation factors depend on budget, jurisdiction, risk tolerance, existing stack, team size, or other user-specific variables]

## Recommendation Basis
[Why one option is favored — or why the decision depends on specific user context]

## Sources
[Numbered list]
```

## High-Stakes (Medical / Legal / Financial)

```markdown
# [Topic]

## Scope
[What this research covers and does not cover]

## TL;DR
[Summary of findings with explicit confidence levels]

## Findings
[Organized by theme. Every claim cites source with date. Conflicting sources are presented side by side.]

## Key Uncertainties
[What remains unclear, debated, or insufficiently sourced]

## Limitations
- This research is based on publicly available sources as of [date]
- [Specific gaps: e.g., "No access to paywalled clinical studies"]
- [Recency issues: e.g., "Most recent source is from [date]"]

## Disclaimer
This is a research summary based on publicly accessible sources, not professional advice, diagnosis, legal interpretation, or investment guidance. Consult a qualified professional before making decisions based on this information.

## Sources
[Numbered list with publication dates]
```
