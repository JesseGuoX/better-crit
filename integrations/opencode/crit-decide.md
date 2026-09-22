---
description: Collect structured human decisions with Crit
agent: build
---

# Collect decisions with Crit

Use this workflow for structured choices in Crit. Ordinary inline review uses
`crit`; this workflow uses `crit decide` and the result rules below.

## Prepare the checklist

Run `crit decide --guide` and follow the installed binary's schema. Use the same
Crit executable for the guide and the session. If it does not support `decide`,
use the project's built binary when available or report that Crit needs updating.

Use a supplied checklist, or write one from the decisions in the current task:
- Give each item enough context, concrete options and meaningful tradeoffs.
- Include a recommendation and reason when justified. Recommendations are not
  user selections.
- Keep checklist, item and option IDs stable across rounds. Do not derive new IDs
  just because a title changes.
- Use JSON for the current version. Descriptions and context may contain
  Markdown; a standalone Markdown input requires explicit support in the guide.
- Store generated input in a task-specific temporary file unless the user names
  a destination. Retain it for recovery and later rounds.

## Open the page and wait

From the target project directory, run:

```bash
crit decide /path/to/decisions.json
```

Relay the actual page URL printed on stderr. Explain that the user can select
options, request changes, or submit with items still pending.

Wait for the command to finish using the shell tool's running-task mechanism.
Do not infer a result from saved drafts, a closed browser, or a disconnected
client. Do not submit on the user's behalf or edit the stored result.

## Interpret the submitted result

Parse stdout as JSON. Exit 0 means a submission was received, including a partial
one. Check `completed` and every item's `status`, `selected`, and `feedback`:

| Status | Agent action |
| --- | --- |
| `decided` | Treat the selected option IDs as constraints for the authorized task. |
| `revision_requested` | Revise the proposal using the feedback, then obtain another decision. Selected options are guidance, not approval. |
| `pending` | Keep the question open. Do not adopt the recommendation or infer agreement. |

Only `completed: true` means every item is decided. Continue work within the
user's authorized scope. For a partial submission, proceed with independent
work supported by decided items; defer work that depends on unanswered items.
Do not apply the inline-review rule that zero comments means approval.

## Continue or recover

Update the same checklist and rerun it from the same project directory when
another decision is needed. Unchanged submitted decisions carry forward;
changed items reset. Changing the checklist title or context invalidates all
items, so keep those stable unless the shared constraints actually change.

Identical input reconnects to the current round or returns its saved result.
Use `crit decide --new-round /path/to/decisions.json` only when intentionally
reopening an unchanged checklist. Do not repeatedly reopen a partial or
all-pending submission without new context or a reason to ask again.

After interruption, rerun the same input to recover. A nonzero exit or malformed
output is a failure to obtain a decision, never approval. Do not switch to the
ordinary review loop, `crit comments`, or approval hooks to complete a decision.

User request or checklist path: $ARGUMENTS
