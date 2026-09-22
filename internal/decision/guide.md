# Crit Plus decision checklists

Use `crit-plus decide checklist.json` (or `crit-plus decide -` for stdin) to ask a human
for structured choices. Run it in the project directory and wait for stdout.
The browser may submit at any time, including with every item pending.
Normal output is exactly one JSON object; logs and the URL go to stderr.

Example input:

```json
{
  "id": "project-storage",
  "title": "Choose a storage approach",
  "context": "The first release runs on one computer.",
  "items": [
    {
      "id": "database",
      "title": "Where should we store data?",
      "context": "We need transactions and offline access.",
      "type": "single",
      "options": [
        {"id": "sqlite", "label": "SQLite", "description": "**Simple deployment.** One local file; limited concurrent writers."},
        {"id": "postgres", "label": "PostgreSQL", "description": "More concurrency; requires a database service."}
      ],
      "recommended": ["sqlite"],
      "recommendation_reason": "Fits the single-computer scope."
    },
    {
      "id": "exports",
      "title": "Which exports should the first release support?",
      "type": "multiple",
      "options": [
        {"id": "json", "label": "JSON"},
        {"id": "csv", "label": "CSV"},
        {"id": "none", "label": "No exports yet"}
      ],
      "recommended": ["json"]
    }
  ]
}
```

Required: checklist `id`, `title`, nonempty `items`; each item has a stable `id`,
`title`, `type` (`single` or `multiple`), and nonempty `options` (`id`, `label`,
optional `description`). Optional `context`, option `description`, and
`recommendation_reason` accept sanitized Markdown. `recommended` contains
option IDs (at most one for single choice). Recommendations are never selected
by default. JSON is limited to 2 MB; unknown fields and duplicate IDs are rejected.

The JSON result contains `session_id`, `checklist_id`, `revision`,
`submission_id`, `submitted_at`, `completed`, and every item's `id`, `status`,
`selected` option IDs and `feedback`:

- `decided`: an explicit submitted selection. Treat it as a constraint.
- `revision_requested`: the human entered feedback. Revise the item and ask
  again. Any selection on this item is guidance for revision, not approval.
- `pending`: no decision. Do not infer agreement from silence or a recommendation.

Only `completed: true` means every item is decided. Exit 0 means a submission
was received, including partial submissions. Never equate exit 0 with approval.
A multiple-choice item with no selection is pending; include a "None" option
if that is a meaningful explicit choice.

Keep the checklist ID, item IDs and option IDs stable across rounds. Update the
same JSON and rerun the command from the same project directory. Changed items
reset to pending. Unchanged submitted decisions carry forward; changing the
checklist title/context invalidates all decisions. Unsubmitted drafts do not
carry into a new version. Removed items remain in saved history.
An identical input reconnects to the current round, or returns its saved result.
Use `--new-round` to reopen an unchanged checklist for pending items.

Drafts save automatically but only Submit releases the waiting agent. The
submitted page is read-only until the agent sends the next version. Persisted
history and snapshots live in `~/.crit/decisions/<session_id>/state.json`.
Closing a tab, disconnecting, or pressing Ctrl+C is not approval. Rerun the
same input after interruption to recover. `crit-plus stop` stops local daemons.

Use `--no-open` for headless use, `--port N` to select a loopback port. This
mode is local and single-user. It does not execute changes, agent commands,
review approval hooks, or automatic review cleanup. The agent interprets the
result and continues only within the user's chosen scope.
