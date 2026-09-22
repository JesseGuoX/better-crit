# crit+ Integrations

crit+ (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Drop-in configuration files that teach your AI coding tool to use crit+ for reviewing plans and code changes and collecting human decisions.

## Marketplace details

The public brand is **crit+**; plugin IDs, executable names, and skills use
`crit-plus`. Each plugin page includes usage instructions and credits the original
Crit project:

| Platform | Plugin details | Marketplace source |
| --- | --- | --- |
| Claude Code | [crit+ for Claude Code](claude-code/README.md) | [Repository marketplace](../.claude-plugin/marketplace.json) |
| Cursor | [crit+ for Cursor](cursor/README.md) | [Repository marketplace](../.cursor-plugin/marketplace.json) |
| Codex | [crit+ for Codex](codex/plugin/crit-plus/README.md) | Local marketplace registered by `crit-plus install codex-plugin` |

## Quick install

```bash
crit-plus install <tool>     # Install for a specific tool in the current project
crit-plus install all        # Install for all supported tools
```

Safe to re-run. Existing files are skipped (use `--force` to overwrite).

**Global install**: run `cd ~ && crit-plus install <tool>` to install to your home directory. The integration is then available across all projects without per-project setup. Each tool reads from a different global path; `crit-plus install` routes the files to the right place automatically.

| Tool | Install command | Project destination | Global destination |
|------|----------------|---------------------|--------------------|
| Claude Code | `crit-plus install claude-code` | `.claude/skills/crit-plus/SKILL.md` + `crit-plus-cli` + `crit-plus-story` + `crit-plus-decide` | `~/.claude/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Cursor | `crit-plus install cursor` | `.cursor/skills/crit-plus/SKILL.md` + `crit-plus-cli` + `crit-plus-story` + `crit-plus-decide` | (project only — Cursor has no stable user-level config dir) |
| GitHub Copilot | `crit-plus install github-copilot` | `.github/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.agents/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| OpenCode | `crit-plus install opencode` | `.opencode/commands/crit-plus.md` + `crit-plus-story.md` + `crit-plus-decide.md` + skills + plugin | `~/.config/opencode/commands/` + `~/.agents/skills/` + plugins |
| Codex | `crit-plus install codex` | `.agents/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.agents/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Codex plugin | `crit-plus install codex-plugin` | loose skills + marketplace + `plugins/crit-plus/` (incl. crit-plus-story and crit-plus-decide) | `~/.agents/` + `~/.codex/plugins/crit-plus/` |
| Pi | `crit-plus install pi` | `.pi/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.pi/agent/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Qwen Code | `crit-plus install qwen` | `.qwen/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.qwen/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Hermes | `crit-plus install hermes` | `.hermes/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` (add `.hermes/skills` to `external_dirs`) | `~/.hermes/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Windsurf | `crit-plus install windsurf` | `.windsurf/workflows/crit-plus.md` + `crit-plus-story.md` + `crit-plus-decide.md` + skills | `~/.codeium/windsurf/global_workflows/` + skills |
| Cline | `crit-plus install cline` | `.clinerules/workflows/crit-plus.md` + `crit-plus-story.md` + `crit-plus-decide.md` + skills | `~/.cline/data/workflows/` + `~/.cline/skills/` |
| Aider | `crit-plus install aider` | `.crit/crit-plus-aider-conventions.md` + adds entry under `read:` in `.aider.conf.yml` | `~/.crit-plus-conventions.md` + adds entry under `read:` in `~/.aider.conf.yml` |
| Gemini CLI | `crit-plus install gemini` | `crit-plus-cli` + `crit-plus-story` + `crit-plus-decide` skills + `/crit-plus` + `/crit-plus-story` + `/crit-plus-decide` commands + policy | same under `~/.gemini/` |
| Grok | `crit-plus install grok` | `.grok/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.grok/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |
| Amp | `crit-plus install ampcode` | `.agents/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` | `~/.config/agents/skills/crit-plus{,-cli,-story,-decide}/SKILL.md` |

## Plugin marketplace (Claude Code)

For the full experience, install via the plugin marketplace. This gives you:
- A `/crit-plus` slash command for the review loop
- A model-discoverable `crit-plus-cli` skill for review files, `crit-plus comment`, `crit-plus pull/push`, etc.
- A wording-gated `/crit-plus-story` skill for chaptered diff overviews that then continues the `/crit-plus` review loop
- A `/crit-plus-decide` skill for decision checklists, partial submissions, and revision rounds

```
claude plugin marketplace add JesseGuoX/crit-plus
claude plugin install crit-plus@crit-plus
```

The marketplace manifest lives at the repo root (`.claude-plugin/marketplace.json`) and points to the plugin files in `integrations/claude-code/`.

### `crit-plus install` vs plugin marketplace

| | `crit-plus install` | Plugin marketplace |
|---|---|---|
| **Scope** | Per-project (committed to repo) | Global (user-wide) |
| **What's installed** | `crit-plus`, `crit-plus-cli`, `crit-plus-story`, and `crit-plus-decide` skills or equivalent workflows | The same skills, plus lifecycle hooks |
| **Good for** | Teams — everyone gets the integration | Individual users — works across all projects |
| **Setup** | Run once per project | Install once, works everywhere |

Both approaches teach the review cycle, CLI operations, story reviews, and structured decisions. The plugin marketplace also installs lifecycle hooks for plan review.

## Claude Code plan approval mode

The Claude Code plugin intercepts `ExitPlanMode` with a narrowly matched
`PermissionRequest` hook. By default, approving in crit+ allows the plan exit and
leaves Claude Code to restore its existing permission mode. To choose the mode
deterministically, set `plan_approve_mode` in your global crit+ config:

```json
{
  "plan_approve_mode": "acceptEdits"
}
```

Disable automatic plan review per shell or globally with
`export CRIT_PLAN_REVIEW=off`. This only disables the plan-exit hook; manually
running `crit-plus plan` still opens a review.

Supported values are `default`, `manual`, `acceptEdits`, `plan`, `auto`,
`dontAsk`, and `bypassPermissions`. The `manual` alias requires Claude Code
2.1.200 or newer. On approval, crit+ returns Claude Code's documented
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

`crit-plus install opencode` also writes a small TypeScript plugin (`crit-plus.ts`) and registers it in `opencode.jsonc`. The plugin shells out to `crit-plus config` on each chat turn and appends sharing instructions to the system prompt only when `share_url` is set. With `share_url: ""` the sharing block is omitted entirely — useful in environments with strict information-sharing policies, and saves tokens otherwise. opencode auto-loads `.ts` files dropped into the plugin directory, so the registration entry is informational.

## Codex plugin

For the full Codex experience, install the plugin. This gives you:

- A `$crit-plus` skill for the review loop (plus loose copies under `.agents/skills/` so bare `$crit-plus` works even outside the plugin)
- A `crit-plus-cli` skill that auto-activates when working with review files, `crit-plus comment`, `crit-plus pull/push`, etc.
- `$crit-plus-story` and `$crit-plus-decide` skills for story reviews and structured human decisions
- A **proposed-plan review hook** — intercepts Codex's `Stop` hook when the agent proposes a plan in Plan mode, writes it to disk, and opens crit+ for inline review before the turn ends

```bash
cd ~ && crit-plus install codex-plugin    # global (recommended)
crit-plus install codex-plugin            # per-project (commit plugins/crit-plus/ for the whole team)
```

`crit-plus install codex-plugin` registers the plugin in a local Codex marketplace (`.agents/plugins/marketplace.json` or `~/.agents/plugins/marketplace.json`), copies plugin files to `plugins/crit-plus/` (project) or `~/.codex/plugins/crit-plus/` (global), enables the plugin in `~/.codex/config.toml`, and turns on `features.plugins`, `features.hooks`, and `features.plugin_hooks`.

Plugin source files live in `integrations/codex/plugin/crit-plus/`. See [`integrations/codex/README.md`](./codex/README.md) for layout and manual setup.

### `crit-plus install codex` vs `crit-plus install codex-plugin`

| | `crit-plus install codex` | `crit-plus install codex-plugin` |
|---|---|---|
| **Scope** | Skills only (project or global) | Skills + Codex plugin + plan hook |
| **What's installed** | `crit-plus`, `crit-plus-cli`, `crit-plus-story`, and `crit-plus-decide` skills under `.agents/skills/` | Same skills, plus `plugins/crit-plus/` (or `~/.codex/plugins/crit-plus/`) with bundled skills and a `Stop` hook |
| **Plan mode** | Agent must write the plan to a file before `$crit-plus` works | Hook captures in-chat proposed plans (`<proposed_plan>`) and reviews them automatically |

Both approaches install all four skills. Only `codex-plugin` adds the proposed-plan hook — without it, typing `$crit-plus` on an in-chat plan (e.g. after choosing "No and stay in Plan Mode") does nothing useful because there is no file path for `crit-plus` to open.

Disable automatic plan review per shell or globally with
`export CRIT_PLAN_REVIEW=off`. Manual `crit-plus plan` invocations are unaffected.

## Invocation policy

The interactive `crit-plus` review cycle is wording-gated, not hard-disabled.
Skill descriptions and workflow docs tell agents to launch crit+ only when the
user invokes the platform command (`/crit-plus`, `$crit-plus`, `/skill:crit-plus`, `/crit-plus.md`,
or Windsurf's `/crit-plus`, as appropriate) or directly asks to use crit+. A normal
request to review code, a plan, a diff, a PR, or a page does not count.

`crit-plus-cli` is intentionally model-discoverable. It teaches agents how to leave
and reply to crit+ comments, interpret review JSON, share reviews, and synchronize
GitHub PR feedback, but it does not start the interactive review cycle.

`crit-plus-story` is wording-gated like `/crit-plus`: agents author a chaptered story
overview only when the user invokes `/crit-plus-story` (or the tool equivalent) or
directly asks to generate a crit-plus story. After ingest they reconnect with bare
`crit-plus` and continue the same wait → address → next-round cycle. It does not run
as part of a normal review request.

`crit-plus-decide` is discoverable for structured human choices and revision
feedback. The existing `crit-plus` entry point also routes decision requests to it.
It waits for a submitted decision result and does not use inline-review
approval rules.

The Claude Code plugin, Codex plugin, and Gemini CLI integration retain their
existing plan-exit hooks for automatic plan review.

## What these do

All integrations follow the same pattern:

1. **Pick the review target** — current git changes by default, or an explicit file, plan, PR, or commit range when the user names one
2. **Launch crit+** — the agent runs the matching `crit-plus` command to open the review in your browser
3. **Address feedback** — after review, the agent reads the review file to find your inline comments and revises the target
4. **Continue the review loop** — the agent reruns the printed next-round command until you finish with no unresolved comments

Each integration also teaches the agent about:
- **`crit-plus comment`** — leave inline review comments programmatically without opening the browser
- **review file format** — how to read comments, resolve them with threaded replies
- **`crit-plus pull/push`** — sync reviews with GitHub PRs (push supports `--event approve|request-changes|comment`)

## Structured human decisions

`crit-plus install <tool>` installs `crit-plus-decide` alongside the existing skills.
OpenCode, Cline, Windsurf, and Gemini also receive a dedicated command or
workflow; Aider receives the same instructions in its conventions file.
The agent discovers when to use decisions from the skill description and reads
`crit-plus decide --guide` at runtime for the schema and examples.

To refresh an existing installation, use `crit-plus install <tool> --force` and
reload the agent's skills or start a new session. Without `--force`, missing
files are added but existing loose skills and conventions are preserved.
Codex plugin files and its activation cache are refreshed on every plugin
install; `--force` also updates its loose skill copies.

For example, ask “Use crit+ to let me choose between these proposals”, invoke
`$crit-plus-decide` in Codex, or `/crit-plus-decide` in Claude Code. Input is currently
JSON; Markdown is supported within the context and description strings.

For a checklist of choices, agents can use `crit-plus decide --guide` and then run
`crit-plus decide checklist.json` (or `crit-plus decide -` for stdin). Wait for the JSON
result and inspect every item's status: `decided` constrains the work,
`revision_requested` requires revision and another confirmation, and `pending`
provides no authorization. Exit 0 may be a partial submission; check `completed`.
Keep IDs stable across rounds. `crit-plus decide` collects choices and does not run
review approval hooks. The [decision guide](../internal/decision/guide.md)
contains the schema, an example, and recovery rules.
