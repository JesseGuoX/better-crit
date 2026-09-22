# Crit - Manual Review Workflow

Use this workflow only when the user explicitly asks to use Crit. A generic
request to review code, a plan, a diff, a PR, or a page does not authorize
launching Crit.

## Structured decisions with Crit

Use this workflow for structured choices in Crit. Ordinary inline review uses
`crit`; this workflow uses `crit decide` and the result rules below.

### Prepare the checklist

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

### Open the page and wait

From the target project directory, run:

```bash
crit decide /path/to/decisions.json
```

Relay the actual page URL printed on stderr. Explain that the user can select
options, request changes, or submit with items still pending.

Wait for the command to finish using the shell tool's running-task mechanism.
Do not infer a result from saved drafts, a closed browser, or a disconnected
client. Do not submit on the user's behalf or edit the stored result.

### Interpret the submitted result

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

### Continue or recover

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

The remaining sections apply when the user requests inline review.

## Review with Crit

After the user explicitly asks to use Crit, launch it:

```bash
crit $PLAN_FILE                       # Review a specific file
crit                                  # Review all changed files in the repo
crit --pr <num|url>                   # Review a GitHub PR (range mode)
crit --mr <iid|url>                   # Review a GitLab MR (range mode)
crit --range <baseSHA>..<headSHA>     # Review a commit range (range mode)
```

**CRITICAL — you MUST run `crit` and block until it completes.**

`crit` starts the daemon if needed, opens the browser, and blocks until the user clicks "Finish Review". It prints the review URL on startup (e.g. `Started crit daemon at http://localhost:<port>`) — relay that URL verbatim.

- Do NOT proceed until `crit` completes.
- Do NOT ask the user to type anything.
- Do NOT read the review file early.

## After review

When `crit` completes, read **stdout** and follow its instructions. Check **stderr** for `approved: true` or `approved: false`.

Field guidance:
- `quote`: the specific text the reviewer selected — focus changes on the quoted text rather than the whole range.
- `anchor`: full text of the commented lines when placed — locate content by anchor, line numbers may be stale.
- `drifted: true`: original content was removed or heavily rewritten — line numbers are approximate at best.

For each unresolved comment:
1. Revise the referenced file using your edit tools.
2. Reply with what you did: `crit comment --reply-to <id> --author 'Aider' '<what you did>'` (markdown supported).
3. **Never pass `--resolve`** unless the user explicitly asks. Resolving is the reviewer's call.

When replying to multiple comments, use `--json`:

```bash
echo '[
  {"reply_to": "c_a1b2c3", "body": "Fixed"},
  {"reply_to": "c_d4e5f6", "body": "Refactored as suggested"}
]' | crit comment --json --author 'Aider'
```

For multi-paragraph reply bodies, prefer `--file <path>`. Raw newlines inside JSON strings are invalid, and shell heredocs make it easy to slip one in:

```bash
cat > /tmp/replies.json <<'EOF'
[
  {"reply_to": "c_a1b2c3", "body": "Fixed.\n\nDetails: split helper, added null guard."}
]
EOF
crit comment --json --file /tmp/replies.json --author 'Aider'
```

`--file -` reads stdin (same as the default).

## Next round

The finish prompt on stdout includes the command to run again — use it to start a new round.

`crit` automatically signals round-complete, then blocks until the next "Finish Review" click. Only proceed after the user approves (a round finishes with zero comments).

## CLI Reference

### `crit comment`

```bash
crit comment --author 'Aider' '<body>'                       # Review-level
crit comment --author 'Aider' <path> '<body>'                # File-level
crit comment --author 'Aider' <path>:<line> '<body>'         # Line
crit comment --author 'Aider' <path>:<start>-<end> '<body>'  # Line range
crit comment --reply-to <id> --author 'Aider' '<body>'       # Reply (c_… or r_…)
```

Hard rules:
- Always pass `--author 'Aider'`.
- Always single-quote the body — double quotes break on backticks and shell metachars.
- Line numbers reference the file on disk (1-indexed), not diff line numbers.
- Reply bodies support markdown.
- Only pass `--resolve` when the user explicitly asks.

If `crit comment` errors with "comment found in multiple files", disambiguate with `--path src/foo.go`.

### Bulk `--json`

For 3+ comments, prefer `--json` (atomic, single write). Synopsis:

```
crit comment --json [--file <path>] [--author <name>]
```

Stdin form (short single-line bodies):

```bash
echo '[
  {"body": "overall feedback", "scope": "review"},
  {"path": "session.go", "body": "restructure", "scope": "file"},
  {"file": "src/auth.go", "line": 42, "body": "Missing null check"},
  {"file": "src/auth.go", "line": "50-55", "body": "Extract to helper"},
  {"reply_to": "c_a1b2c3", "body": "Fixed — added null check"}
]' | crit comment --json --author 'Aider'
```

`--file <path>` form — preferred whenever a body has paragraph breaks (raw newlines in a JSON string are invalid):

```bash
cat > /tmp/crit-bulk.json <<'EOF'
[
  {"file": "src/auth.go", "line": 42, "body": "Para 1.\n\nPara 2."}
]
EOF
crit comment --json --file /tmp/crit-bulk.json --author 'Aider'
```

Scope inference: `reply_to` → reply; no `file`/`line` → review; `path` only → file; `path` + `line` → line.

### Sharing

```bash
crit share <file> [file...]                          # Upload and print URL
crit share --qr <file>                               # Also print QR code (terminal only)
crit share --org <slug> <file>                       # Share under an organization
crit share --org <slug> --visibility unlisted <file> # Org share with explicit visibility
crit unpublish [file...]                              # Remove shared review
```

Always relay the full output (URL, QR) directly in your response — don't make the user dig through tool output.
- **`--org <slug>`** shares under an organization. Visibility defaults to `organization` (members only). Override with `--visibility` (`organization`, `unlisted`, `public`).

### GitHub PR / GitLab MR sync

```bash
crit pull [number|url]                                   # Fetch PR/MR comments
crit push [--dry-run] [--event <type>] [-m <msg>] [pr]   # Post review as PR review
```

Requires `gh` CLI. `--event`: `comment` (default), `approve`, `request-changes`.
