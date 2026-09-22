# Crit Integrations

Drop-in configuration files that teach your AI coding tool to use Crit for reviewing plans and code changes and collecting human decisions.

## Quick install

```bash
crit install <tool>     # Install for a specific tool in the current project
crit install all        # Install for all supported tools
```

Safe to re-run. Existing files are skipped (use `--force` to overwrite).

**Global install**: run `cd ~ && crit install <tool>` to install to your home directory. The integration is then available across all projects without per-project setup. Each tool reads from a different global path; `crit install` routes the files to the right place automatically.

| Tool | Install command | Project destination | Global destination |
|------|----------------|---------------------|--------------------|
| Claude Code | `crit install claude-code` | `.claude/skills/crit/SKILL.md` + `crit-cli` + `crit-story` + `crit-decide` | `~/.claude/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Cursor | `crit install cursor` | `.cursor/skills/crit/SKILL.md` + `crit-cli` + `crit-story` + `crit-decide` | (project only — Cursor has no stable user-level config dir) |
| GitHub Copilot | `crit install github-copilot` | `.github/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.agents/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| OpenCode | `crit install opencode` | `.opencode/commands/crit.md` + `crit-story.md` + `crit-decide.md` + skills + plugin | `~/.config/opencode/commands/` + `~/.agents/skills/` + plugins |
| Codex | `crit install codex` | `.agents/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.agents/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Codex plugin | `crit install codex-plugin` | loose skills + marketplace + `plugins/crit/` (incl. crit-story and crit-decide) | `~/.agents/` + `~/.codex/plugins/crit/` |
| Pi | `crit install pi` | `.pi/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.pi/agent/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Qwen Code | `crit install qwen` | `.qwen/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.qwen/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Hermes | `crit install hermes` | `.hermes/skills/crit{,-cli,-story,-decide}/SKILL.md` (add `.hermes/skills` to `external_dirs`) | `~/.hermes/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Windsurf | `crit install windsurf` | `.windsurf/workflows/crit.md` + `crit-story.md` + `crit-decide.md` + skills | `~/.codeium/windsurf/global_workflows/` + skills |
| Cline | `crit install cline` | `.clinerules/workflows/crit.md` + `crit-story.md` + `crit-decide.md` + skills | `~/.cline/data/workflows/` + `~/.cline/skills/` |
| Aider | `crit install aider` | `.crit/aider-conventions.md` + adds entry under `read:` in `.aider.conf.yml` | `~/.crit-conventions.md` + adds entry under `read:` in `~/.aider.conf.yml` |
| Gemini CLI | `crit install gemini` | `crit-cli` + `crit-story` + `crit-decide` skills + `/crit` + `/crit-story` + `/crit-decide` commands + policy | same under `~/.gemini/` |
| Grok | `crit install grok` | `.grok/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.grok/skills/crit{,-cli,-story,-decide}/SKILL.md` |
| Amp | `crit install ampcode` | `.agents/skills/crit{,-cli,-story,-decide}/SKILL.md` | `~/.config/agents/skills/crit{,-cli,-story,-decide}/SKILL.md` |

## Plugin marketplace (Claude Code)

For the full experience, install via the plugin marketplace. This gives you:
- A `/crit` slash command for the review loop
- A model-discoverable `crit-cli` skill for review files, `crit comment`, `crit pull/push`, etc.
- A wording-gated `/crit-story` skill for chaptered diff overviews that then continues the `/crit` review loop
- A `/crit-decide` skill for decision checklists, partial submissions, and revision rounds

```
claude plugin marketplace add tomasz-tomczyk/crit
claude plugin install crit@crit
```

The marketplace manifest lives at the repo root (`.claude-plugin/marketplace.json`) and points to the plugin files in `integrations/claude-code/`.

### `crit install` vs plugin marketplace

| | `crit install` | Plugin marketplace |
|---|---|---|
| **Scope** | Per-project (committed to repo) | Global (user-wide) |
| **What's installed** | `crit`, `crit-cli`, `crit-story`, and `crit-decide` skills or equivalent workflows | The same skills, plus lifecycle hooks |
| **Good for** | Teams — everyone gets the integration | Individual users — works across all projects |
| **Setup** | Run once per project | Install once, works everywhere |

Both approaches teach the review cycle, CLI operations, story reviews, and structured decisions. The plugin marketplace also installs lifecycle hooks for plan review.

## Claude Code plan approval mode

The Claude Code plugin intercepts `ExitPlanMode` with a narrowly matched
`PermissionRequest` hook. By default, approving in Crit allows the plan exit and
leaves Claude Code to restore its existing permission mode. To choose the mode
deterministically, set `plan_approve_mode` in your global Crit config:

```json
{
  "plan_approve_mode": "acceptEdits"
}
```

Disable automatic plan review per shell or globally with
`export CRIT_PLAN_REVIEW=off`. This only disables the plan-exit hook; manually
running `crit plan` still opens a review.

Supported values are `default`, `manual`, `acceptEdits`, `plan`, `auto`,
`dontAsk`, and `bypassPermissions`. The `manual` alias requires Claude Code
2.1.200 or newer. On approval, Crit returns Claude Code's documented
`decision.updatedPermissions` entry:

```json
{
  "type": "setMode",
  "mode": "acceptEdits",
  "destination": "session"
}
```

`destination: "session"` keeps the change in memory for the current Claude Code
session only. The setting is global-only (`~/.crit.config.json`); a repository's
`.crit.config.json` cannot change your permission policy. Unset preserves the
default hook behavior, and invalid values are ignored with a warning.

The hook approval and mode switch do not override matching deny or ask rules.
Claude Code can also disable `auto` through `permissions.disableAutoMode`.
`bypassPermissions` is intentionally dangerous and should only be used in an
isolated environment. Claude Code applies it only when the session started with
bypass mode available (for example `--allow-dangerously-skip-permissions` or
`--dangerously-skip-permissions`) and managed settings have not disabled it;
otherwise Claude Code treats the update as a no-op.

## OpenCode plugin: conditional sharing instructions

`crit install opencode` also writes a small TypeScript plugin (`crit.ts`) and registers it in `opencode.jsonc`. The plugin shells out to `crit config` on each chat turn and appends sharing instructions to the system prompt only when `share_url` is set. With `share_url: ""` the sharing block is omitted entirely — useful in environments with strict information-sharing policies, and saves tokens otherwise. opencode auto-loads `.ts` files dropped into the plugin directory, so the registration entry is informational.

## Codex plugin

For the full Codex experience, install the plugin. This gives you:

- A `$crit` skill for the review loop (plus loose copies under `.agents/skills/` so bare `$crit` works even outside the plugin)
- A `crit-cli` skill that auto-activates when working with review files, `crit comment`, `crit pull/push`, etc.
- `$crit-story` and `$crit-decide` skills for story reviews and structured human decisions
- A **proposed-plan review hook** — intercepts Codex's `Stop` hook when the agent proposes a plan in Plan mode, writes it to disk, and opens Crit for inline review before the turn ends

```bash
cd ~ && crit install codex-plugin    # global (recommended)
crit install codex-plugin            # per-project (commit plugins/crit/ for the whole team)
```

`crit install codex-plugin` registers the plugin in a local Codex marketplace (`.agents/plugins/marketplace.json` or `~/.agents/plugins/marketplace.json`), copies plugin files to `plugins/crit/` (project) or `~/.codex/plugins/crit/` (global), enables the plugin in `~/.codex/config.toml`, and turns on `features.plugins`, `features.hooks`, and `features.plugin_hooks`.

Plugin source files live in `integrations/codex/plugin/crit/`. See [`integrations/codex/README.md`](./codex/README.md) for layout and manual setup.

### `crit install codex` vs `crit install codex-plugin`

| | `crit install codex` | `crit install codex-plugin` |
|---|---|---|
| **Scope** | Skills only (project or global) | Skills + Codex plugin + plan hook |
| **What's installed** | `crit`, `crit-cli`, `crit-story`, and `crit-decide` skills under `.agents/skills/` | Same skills, plus `plugins/crit/` (or `~/.codex/plugins/crit/`) with bundled skills and a `Stop` hook |
| **Plan mode** | Agent must write the plan to a file before `$crit` works | Hook captures in-chat proposed plans (`<proposed_plan>`) and reviews them automatically |

Both approaches install all four skills. Only `codex-plugin` adds the proposed-plan hook — without it, typing `$crit` on an in-chat plan (e.g. after choosing "No and stay in Plan Mode") does nothing useful because there is no file path for `crit` to open.

Disable automatic plan review per shell or globally with
`export CRIT_PLAN_REVIEW=off`. Manual `crit plan` invocations are unaffected.

## Invocation policy

The interactive `crit` review cycle is wording-gated, not hard-disabled.
Skill descriptions and workflow docs tell agents to launch Crit only when the
user invokes the platform command (`/crit`, `$crit`, `/skill:crit`, `/crit.md`,
or Windsurf's `/crit`, as appropriate) or directly asks to use Crit. A normal
request to review code, a plan, a diff, a PR, or a page does not count.

`crit-cli` is intentionally model-discoverable. It teaches agents how to leave
and reply to Crit comments, interpret review JSON, share reviews, and synchronize
GitHub PR feedback, but it does not start the interactive review cycle.

`crit-story` is wording-gated like `/crit`: agents author a chaptered story
overview only when the user invokes `/crit-story` (or the tool equivalent) or
directly asks to generate a crit story. After ingest they reconnect with bare
`crit` and continue the same wait → address → next-round cycle. It does not run
as part of a normal review request.

`crit-decide` is discoverable for structured human choices and revision
feedback. The existing `crit` entry point also routes decision requests to it.
It waits for a submitted decision result and does not use inline-review
approval rules.

The Claude Code plugin, Codex plugin, and Gemini CLI integration retain their
existing plan-exit hooks for automatic plan review.

## What these do

All integrations follow the same pattern:

1. **Pick the review target** — current git changes by default, or an explicit file, plan, PR, or commit range when the user names one
2. **Launch Crit** — the agent runs the matching `crit` command to open the review in your browser
3. **Address feedback** — after review, the agent reads the review file to find your inline comments and revises the target
4. **Continue the review loop** — the agent reruns the printed next-round command until you finish with no unresolved comments

Each integration also teaches the agent about:
- **`crit comment`** — leave inline review comments programmatically without opening the browser
- **review file format** — how to read comments, resolve them with threaded replies
- **`crit pull/push`** — sync reviews with GitHub PRs (push supports `--event approve|request-changes|comment`)

## Structured human decisions

`crit install <tool>` installs `crit-decide` alongside the existing skills.
OpenCode, Cline, Windsurf, and Gemini also receive a dedicated command or
workflow; Aider receives the same instructions in its conventions file.
The agent discovers when to use decisions from the skill description and reads
`crit decide --guide` at runtime for the schema and examples.

To refresh an existing installation, use `crit install <tool> --force` and
reload the agent's skills or start a new session. Without `--force`, missing
files are added but existing loose skills and conventions are preserved.
Codex plugin files and its activation cache are refreshed on every plugin
install; `--force` also updates its loose skill copies.

For example, ask “Use Crit to let me choose between these proposals”, invoke
`$crit-decide` in Codex, or `/crit-decide` in Claude Code. Input is currently
JSON; Markdown is supported within the context and description strings.

For a checklist of choices, agents can use `crit decide --guide` and then run
`crit decide checklist.json` (or `crit decide -` for stdin). Wait for the JSON
result and inspect every item's status: `decided` constrains the work,
`revision_requested` requires revision and another confirmation, and `pending`
provides no authorization. Exit 0 may be a partial submission; check `completed`.
Keep IDs stable across rounds. `crit decide` collects choices and does not run
review approval hooks. The [decision guide](../internal/decision/guide.md)
contains the schema, an example, and recovery rules.
