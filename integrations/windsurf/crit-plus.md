# Crit Plus - Review Agent Output

Crit Plus (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

## Choose the workflow

If the user wants to select proposals, adjudicate questions, or supplies
`decide` arguments, load the installed `crit-plus-decide` skill. It prepares the
checklist, runs `crit-plus decide`, and handles partial submissions and revisions.
Use that workflow through completion; the remaining steps here apply to inline
review, whose approval rules differ from decision results.

Before implementing any non-trivial feature, write an implementation plan as a markdown file.

## Writing plans

When asked to implement a feature, first create a plan file that covers:
- What will be built
- Which files will be created or modified
- Key design decisions and trade-offs
- Step-by-step implementation order

## Review with Crit Plus

After writing a plan or code, launch Crit Plus:

```bash
crit-plus $PLAN_FILE                       # Review a specific file
crit-plus                                  # Review all changed files in the repo
crit-plus --pr <num|url>                   # Review a GitHub PR (range mode)
crit-plus --mr <iid|url>                   # Review a GitLab MR (range mode)
crit-plus --range <baseSHA>..<headSHA>     # Review a commit range (range mode)
```

**CRITICAL — you MUST run `crit-plus` and block until it completes.**

`crit-plus` starts the daemon if needed, opens the browser, and blocks until the user clicks "Finish Review". It prints the review URL on startup (e.g. `Started crit-plus daemon at http://localhost:<port>`) — relay that URL verbatim.

- Do NOT proceed until `crit-plus` completes.
- Do NOT ask the user to type anything.
- Do NOT read the review file early.

## After review

When `crit-plus` completes, read **stdout** and follow its instructions. Check **stderr** for `approved: true` or `approved: false`.

Field guidance:
- `quote`: the specific text the reviewer selected — focus changes on the quoted text rather than the whole range.
- `anchor`: full text of the commented lines when placed — locate content by anchor, line numbers may be stale.
- `drifted: true`: original content was removed or heavily rewritten — line numbers are approximate at best.

For each unresolved comment:
1. Revise the referenced file using your edit tools.
2. Reply with what you did: `crit-plus comment --reply-to <id> --author 'Windsurf' '<what you did>'` (markdown supported).
3. **Never pass `--resolve`** unless the user explicitly asks. Resolving is the reviewer's call.

When replying to multiple comments, use `--json`:

```bash
echo '[
  {"reply_to": "c_a1b2c3", "body": "Fixed"},
  {"reply_to": "c_d4e5f6", "body": "Refactored as suggested"}
]' | crit-plus comment --json --author 'Windsurf'
```

If any reply body spans multiple paragraphs, write the JSON to a temp file and pass `--file <path>` instead — a raw newline inside a JSON `"body"` string is a parse error, and shell heredocs make it easy to introduce one:

```bash
cat > /tmp/replies.json <<'EOF'
[
  {"reply_to": "c_a1b2c3", "body": "Fixed.\n\nDetails: split helper, added null guard."}
]
EOF
crit-plus comment --json --file /tmp/replies.json --author 'Windsurf'
```

`--file -` is the explicit "read stdin" form.

## Next round

The finish prompt on stdout includes the command to run again — use it to start a new round.

`crit-plus` automatically signals round-complete, then blocks until the next "Finish Review" click. Only proceed after the user approves (a round finishes with zero comments).

## CLI Reference

### `crit-plus comment`

```bash
crit-plus comment --author 'Windsurf' '<body>'                       # Review-level
crit-plus comment --author 'Windsurf' <path> '<body>'                # File-level
crit-plus comment --author 'Windsurf' <path>:<line> '<body>'         # Line
crit-plus comment --author 'Windsurf' <path>:<start>-<end> '<body>'  # Line range
crit-plus comment --reply-to <id> --author 'Windsurf' '<body>'       # Reply (c_… or r_…)
```

Hard rules:
- Always pass `--author 'Windsurf'`.
- Always single-quote the body — double quotes break on backticks and shell metachars.
- Line numbers reference the file on disk (1-indexed), not diff line numbers.
- Reply bodies support markdown.
- Only pass `--resolve` when the user explicitly asks.

If `crit-plus comment` errors with "comment found in multiple files", disambiguate with `--path src/foo.go`.

### Bulk `--json`

For 3+ comments, prefer `--json` (atomic, single write). Synopsis:

```
crit-plus comment --json [--file <path>] [--author <name>]
```

Stdin form (short single-line bodies):

```bash
echo '[
  {"body": "overall feedback", "scope": "review"},
  {"path": "session.go", "body": "restructure", "scope": "file"},
  {"file": "src/auth.go", "line": 42, "body": "Missing null check"},
  {"file": "src/auth.go", "line": "50-55", "body": "Extract to helper"},
  {"reply_to": "c_a1b2c3", "body": "Fixed — added null check"}
]' | crit-plus comment --json --author 'Windsurf'
```

`--file <path>` form — use whenever a body has paragraph breaks, since raw newlines in a JSON string are invalid:

```bash
cat > /tmp/crit-plus-bulk.json <<'EOF'
[
  {"file": "src/auth.go", "line": 42, "body": "Para 1.\n\nPara 2."}
]
EOF
crit-plus comment --json --file /tmp/crit-plus-bulk.json --author 'Windsurf'
```

Scope inference: `reply_to` → reply; no `file`/`line` → review; `path` only → file; `path` + `line` → line.

### Sharing

```bash
crit-plus share <file> [file...]                          # Upload and print URL
crit-plus share --qr <file>                               # Also print QR code (terminal only)
crit-plus share --org <slug> <file>                       # Share under an organization
crit-plus share --org <slug> --visibility unlisted <file> # Org share with explicit visibility
crit-plus unpublish [file...]                              # Remove shared review
```

Always relay the full output (URL, QR) directly in your response — don't make the user dig through tool output.
- **`--org <slug>`** shares under an organization. Visibility defaults to `organization` (members only). Override with `--visibility` (`organization`, `unlisted`, `public`).

### GitHub PR / GitLab MR sync

```bash
crit-plus pull [number|url]                                   # Fetch PR/MR comments
crit-plus push [--dry-run] [--event <type>] [-m <msg>] [pr]   # Post review as PR review
```

Requires `gh` CLI. `--event`: `comment` (default), `approve`, `request-changes`.
