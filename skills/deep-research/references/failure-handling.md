# Failure and Degradation Handling

## Scrape Failures

| Failure Type | Action |
|-------------|--------|
| **Timeout / connection error** | Retry once. If still fails, search for the same info from an alternative URL. |
| **Paywall / login required** | Do not retry. Search for the same content from open-access sources, or use the search snippet. If the paywalled source appears central to the question, note that the strongest known source could not be reviewed in full and reduce confidence accordingly. |
| **JavaScript-only rendering** | Scrape will likely return empty or minimal content. Use the search snippet and note it. |
| **PDF / non-HTML** | Scrape may fail or return garbled text. Note the gap and search for an HTML version of the same content. |
| **Redirect loop / 4xx / 5xx** | Skip. Find alternative source. |
| **Duplicate content** | If the same article appears on multiple domains (syndication), scrape only once from the original publisher. |

## Information Gaps

| Situation | Action |
|-----------|--------|
| **Too few search results** | Broaden the query. Try synonyms, related terms, or English if the original language yielded little. |
| **All scrapes failed for a subtopic (low-stakes)** | Fall back to search snippets. Clearly mark affected claims as `[based on search excerpt]`. |
| **All scrapes failed for a subtopic (high-stakes)** | Do not synthesize substantive conclusions from snippets alone. Narrow the output to what is clearly established, what could not be verified, and what requires professional review or direct source inspection. |
| **No authoritative sources found** | State this explicitly. Do not elevate blog posts or forums to fill the gap. |
| **Key information is behind paywall** | Note the source exists but could not be accessed. Cite the title and URL for the user to check manually. |
| **Only outdated sources available** | Use them for background but flag the date prominently. State that current information may differ. |

## Conflicting Information

| Situation | Action |
|-----------|--------|
| **Two credible sources disagree** | Present both positions with their sources. Do not pick a winner unless one is clearly more authoritative and recent. |
| **One source contradicts many** | Note the outlier. Check if it's more recent, more authoritative, or simply wrong. |
| **Numbers/stats differ across sources** | Prefer the primary data source (e.g., the original study, official statistics). Note the discrepancy. |
| **Unverifiable claim from single source** | Include it only if relevant, tagged as `[single source, unverified]`. Never present it as established fact. |

## Graceful Degradation Levels

1. **Full depth**: Multiple authoritative sources scraped and cross-verified. High confidence.
2. **Partial depth**: Mix of full scrapes and snippets. Medium confidence. Note which sections are snippet-based.
3. **Snippet-only**: No successful scrapes. Report is based entirely on search result excerpts. Low confidence. State this upfront.

Always be transparent about which level the final report achieved.

## High-Stakes Escalation Rules

For medical, legal, financial, or regulatory topics:
- Do not rely on snippet-only evidence for actionable conclusions
- If only weak or partial evidence is available, downgrade the output to a scoped research summary
- Explicitly separate:
  - Verified findings from accessible authoritative sources
  - Unresolved questions where evidence is insufficient
  - Inaccessible or unverified primary sources the user may want to review directly
- Never present uncertain findings as settled consensus in high-stakes domains
