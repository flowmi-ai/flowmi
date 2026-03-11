---
name: deep-research
description: >
  Adaptive deep research on any topic — background research, landscape analysis, due diligence,
  vendor/tool/company comparison, trend reports, briefing memos, technology investigation,
  policy research, or any question requiring thorough multi-source analysis.
  Use this skill whenever the user asks to research something, investigate a topic, compare options
  in depth, find out what's happening with X, do a deep dive, write a briefing, or asks any question
  that would benefit from searching multiple sources rather than answering from memory alone.
  Also trigger when the user says "look into", "what's the latest on", "give me a rundown",
  or any variant of "research this for me".
---

# Deep Research

An adaptive research workflow powered by `flowmi search` and `flowmi scrape`. This is not a fixed script — it's a methodology that adjusts depth, breadth, source strategy, and output format based on the topic.

## Prerequisites

The user must have `flowmi` installed and authenticated (`flowmi auth login`).

## 1. Research Triage

Before searching anything, classify the task along these dimensions:

| Dimension | Options |
|-----------|---------|
| **Topic type** | general / technical / company / academic / medical / legal / financial / policy |
| **Timeliness** | timeless / recent / breaking |
| **Geography** | global / country-specific / jurisdiction-specific |
| **Stakes** | low (informational) / high (decision-driving, health, legal, financial) |
| **Output goal** | brief answer / comparison / due diligence / trend report / decision memo |

Perform triage internally before researching. Only surface the triage explicitly when it affects the answer, such as:
- Jurisdiction or geography assumptions
- Date cutoffs or freshness constraints
- High-stakes limitations or disclaimers
- Scope decisions that materially affect the conclusion

## 2. Search Planning

Generate search queries based on the triage. Do not use a fixed number — use as many as needed to cover the core angles of the topic.

**Guidelines:**
- Simple factual question: 2-3 queries may suffice
- Broad landscape analysis: 5-8 queries across sub-topics
- Controversial or multi-stakeholder topic: include queries for each major perspective
- For country/jurisdiction-specific topics: add locale flags (`--country`, `--language`)
- For recent/breaking topics: pair web searches with news searches

**Commands:**

```bash
flowmi search "<query>" -o json -L 10
flowmi search news "<query>" -o json -L 10 --time week
```

Run all initial queries in parallel.

## 3. Source Selection

Select URLs to scrape based on the topic type. See [references/source-quality.md](references/source-quality.md) for the full hierarchy.

**Core rules:**
- **Technical**: official docs, specs, repos, maintainer posts first
- **Company/business**: official site, filings, investor relations, product docs first
- **Academic**: papers, preprints, institutional reports first
- **Medical**: guidelines, systematic reviews, regulatory agencies first
- **Legal/policy**: statutes, regulatory bodies, court decisions, official notices first
- **News events**: original reporting, party statements, cross-verify across multiple outlets

**Source verification by claim type** — see [references/claim-types.md](references/claim-types.md):
- For factual claims governed by a single authoritative source, prefer that primary source directly
- For interpretive, comparative, disputed, or impact-related claims, require 2+ independent sources
- If only one non-authoritative source supports a claim, label it low confidence
- Prefer diverse domains — avoid scraping 3+ pages from the same site
- For high-stakes topics, demand authoritative primary sources, not blog summaries

## 4. Scrape and Read

```bash
flowmi scrape "<url>" -o json
```

Scrape in parallel. Use the `markdown` field (preferred over `text`).

**Failure handling** — see [references/failure-handling.md](references/failure-handling.md):
- If scrape fails (paywall, JS-only, timeout): search for the same information from an alternative source
- If a paywalled source appears central to the question, note that the strongest known source could not be reviewed in full and reduce confidence accordingly
- If still unavailable: use the search snippet but tag it as `[snippet-only, not verified]`
- Skip duplicate content (same article syndicated across sites)
- For PDFs or non-HTML content that fails to scrape: note the gap explicitly

## 5. Iterative Deepening

After reading scraped content, assess coverage:

- **Gaps found?** Generate refined follow-up queries targeting what's missing. Search and scrape again.
- **Conflicting claims?** Search specifically to resolve the conflict — look for primary sources or more authoritative references.
- **Sufficient coverage?** Stop. Do not search for the sake of searching.

**Stop when any of the following is true:**
- The main angles from triage are covered
- Additional searches return mostly repetitive results
- Only low-value or duplicate sources remain
- The remaining uncertainty cannot be resolved with publicly accessible sources

For simple topics this may be 1 round; for complex ones, 3-4. Never exceed 4 rounds of iterative deepening. If coverage is still insufficient after 4 rounds, move to synthesis and note the gaps.

## 6. Evidence Ledger

Before writing the report, organize your findings into a structured evidence layer. For each key claim, track:

- **Claim**: what is being asserted
- **Claim type**: factual / interpretive / comparative / predictive — see [references/claim-types.md](references/claim-types.md)
- **Source**: URL
- **Source type**: official docs / peer-reviewed / news / blog / forum / snippet-only
- **Date**: publication date (if available)
- **Authority**: authoritative primary / credible secondary / weak secondary / snippet-only
- **Confidence**: high (primary source, verified) / medium (credible secondary) / low (single source, snippet-only, or outdated)
- **Conflicts**: note if other sources contradict this

Example:

| Claim | Type | Source | Authority | Confidence | Conflicts |
|-------|------|--------|-----------|------------|-----------|
| Fly.io supports GPU workloads | factual | [Fly.io docs](https://fly.io/docs/gpus/) | authoritative primary | high | — |
| Railway's cold starts are slower than Fly.io's | comparative | [blog post](https://example.com/comparison) | credible secondary | medium | One user reports opposite on Reddit |
| Fly.io will add EU regions in Q3 | predictive | [founder tweet](https://x.com/example) | weak secondary | low | — |

You do not need to print this ledger in the final output, but you must build it internally before synthesizing. For high-stakes topics, include a summary version in the report.

## 7. Synthesis

Choose the output template based on your triage. See [references/output-templates.md](references/output-templates.md) for full templates.

| Output goal | Template |
|-------------|----------|
| **Brief answer / explainer** | TL;DR + Key Findings + Sources |
| **Comparison** | Criteria + Side-by-Side + Trade-offs + Evidence Quality Notes |
| **Trend report** | Timeline + Recent Developments + Outlook |
| **Decision memo** | Options + Pros/Cons + Risks + Context Dependencies |
| **High-stakes** | Scope + Findings + Uncertainties + Limitations + Disclaimer |

**Synthesis rules:**
- Organize by theme, not by source
- Cite every claim inline with `[Source Name](url)`
- Separate established background from recent changes
- If sources conflict on a key point, present all sides with their sources — do not force a single narrative
- For time-sensitive topics, include an "As of [date]" line near the top and note publication dates of key sources
- For high-stakes topics (medical, legal, financial), always end with a limitations section and a disclaimer that this is research, not professional advice

## 8. Failure and Degradation

See [references/failure-handling.md](references/failure-handling.md) for detailed handling of scrape failures, information gaps, conflicting sources, and graceful degradation levels. The key principles:

- Match response quality to source quality — do not overstate confidence when evidence is thin
- For high-stakes topics, prefer narrowing the answer over synthesizing from weak sources
- Always be transparent about the degradation level the final report achieved (full depth / partial depth / snippet-only)

## Example Workflow

User asks: *"Compare Fly.io vs Railway for deploying Go APIs"*

1. **Triage**: technical + company, timeless (mostly), global, medium stakes, comparison output
2. **Search** (4 queries in parallel):
   - `flowmi search "Fly.io vs Railway Go deployment" -o json -L 10`
   - `flowmi search "Fly.io Go API hosting review 2025" -o json -L 10`
   - `flowmi search "Railway Go deployment experience" -o json -L 10`
   - `flowmi search "Fly.io Railway pricing comparison" -o json -L 10`
3. **Source selection**: pick official docs (fly.io/docs, docs.railway.app), pricing pages, 2-3 practitioner comparison posts, 1 migration case study — 6-8 URLs total
4. **Scrape** all selected URLs in parallel
5. **Assess coverage**: pricing covered, DX covered, but cold start performance data is thin → 1 follow-up query: `flowmi search "Fly.io Railway cold start latency benchmark" -o json -L 5`
6. **Evidence ledger**: build internally, noting that cold start data comes from a single blog post (medium confidence)
7. **Synthesize**: use the Comparison template — TL;DR, criteria table, trade-offs, evidence quality notes, sources

Total: 5 searches, ~8 scrapes, 2 rounds. Report runs ~80 lines, output inline.

## Research Integrity Rules

- Do not present inaccessible, snippet-only, or weakly sourced material as established fact
- Match evidentiary standards to the topic's stakes
- Prefer narrowing the answer over overstating confidence
- Separate verified facts, informed interpretation, and unresolved uncertainty
- When a single authoritative primary source governs the answer, use it directly rather than forcing artificial source-count requirements

## 9. Follow-Up Questions

When the user asks a follow-up ("tell me more about X", "dig deeper into the pricing", "what about Option C?"):

- **Targeted follow-up**: run 1-2 new searches focused on the specific question, scrape relevant results, and integrate with findings already gathered. No need to redo the full workflow.
- **Scope expansion**: if the follow-up broadens the topic significantly (e.g., adding a new option to a comparison), run a mini version of the full workflow for the new scope and merge into the existing report.
- **Clarification**: if the user asks about something already covered, reference the existing findings — do not re-search unless they indicate the answer was wrong or incomplete.

## Flowmi Command Reference

Use the commands below as building blocks. Do not let command execution patterns override source quality, research judgment, or scope discipline.

Adjust `-L` based on topic breadth: 5 for focused factual queries, 10 for standard research, up to 20 for broad landscape scans.

| Command | Purpose |
|---------|---------|
| `flowmi search "<q>" -o json -L 10` | Web search, 10 results |
| `flowmi search news "<q>" -o json -L 10 --time week` | News search, past week |
| `flowmi search news "<q>" -o json -L 10 --time month` | News search, past month |
| `flowmi search "<q>" -o json -L 10 --country us --language en` | Locale-specific search |
| `flowmi scrape "<url>" -o json` | Full page scrape (returns markdown + text) |

## Output Delivery

- Short reports (under ~100 lines): output directly in the conversation.
- Long reports (comparisons, trend analyses, decision memos): ask the user if they'd like it saved to a file, or output inline if they don't specify.

## Allowed Tools

- Bash (for running `flowmi` commands)
- Read (for local files the user references as input)
- Write (only when the user asks for the report saved to a file)
