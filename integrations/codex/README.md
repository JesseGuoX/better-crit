# crit+ — Codex CLI Integration

crit+ (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Drop-in configuration files that teach the OpenAI Codex CLI to use crit+ for reviewing plans and code changes and collecting human decisions.

See [crit+ for Codex](plugin/crit-plus/README.md) for the marketplace overview,
features, attribution, and example prompts.

## What's included

| Path | Install command | Purpose |
|------|----------------|---------|
| `skills/crit-plus/SKILL.md` | `crit-plus install codex` | `$crit-plus` skill — launches the interactive review loop |
| `skills/crit-plus-cli/SKILL.md` | `crit-plus install codex` | CLI reference — `crit-plus comment`, `crit-plus pull` / `crit-plus push`, review file format |
| `skills/crit-plus-story/SKILL.md` | `crit-plus install codex` | Chaptered diff overview followed by inline review |
| `skills/crit-plus-decide/SKILL.md` | `crit-plus install codex` | Decision checklist — choices, recommendations, revision feedback, and follow-up rounds |
| `plugin/crit-plus/.codex-plugin/plugin.json` | `crit-plus install codex-plugin` | Codex plugin manifest (skills + hooks) |
| `plugin/crit-plus/README.md` | `crit-plus install codex-plugin` | Plugin detail page with usage and upstream attribution |
| `plugin/crit-plus/skills/*` | `crit-plus install codex-plugin` | Plugin-packaged copies of the skills |
| `plugin/crit-plus/hooks/hooks.json` | `crit-plus install codex-plugin` | `Stop` hook → `crit-plus plan-hook --mode codex` for proposed-plan review |

## Install

**Skills only** (on-demand `$crit-plus` when the target is already a file on disk):

```bash
crit-plus install codex              # project: .agents/skills/
cd ~ && crit-plus install codex      # global: ~/.agents/skills/
```

**Full plugin** (recommended — adds proposed-plan review in Plan mode):

```bash
crit-plus install codex-plugin              # project: plugins/crit-plus/ + .agents/skills/
cd ~ && crit-plus install codex-plugin      # global: ~/.codex/plugins/crit-plus/ + ~/.agents/skills/
```

The plugin install also:

1. Registers crit+ in `.agents/plugins/marketplace.json` (project) or `~/.agents/plugins/marketplace.json` (global)
2. Enables `crit-plus@local` in `~/.codex/config.toml`
3. Sets `features.plugins`, `features.hooks`, and `features.plugin_hooks` to `true` in that config

Safe to re-run. Existing loose skills are skipped unless you pass `--force`;
plugin files and the activation cache are refreshed on every plugin install.
Use `crit-plus install codex --force` (or `codex-plugin --force`) to update existing
skills with decision routing. Start a new Codex session to load the updated skills.

## Plan mode and in-chat plans

In Codex Plan mode, the agent often keeps the plan in chat (`<proposed_plan>`) instead of writing a file. Bare `$crit-plus` needs a path, so it cannot review those plans on its own.

With `crit-plus install codex-plugin`, the `Stop` hook reads the proposed plan from the Codex transcript, writes it to a temp file, and runs `crit-plus plan-hook --mode codex`. You leave inline comments, the agent revises, and the turn does not complete until you approve.

Disable the hook: `export CRIT_PLAN_REVIEW=off`

## Usage

Once installed:

- Type `$crit-plus` in Codex chat to review current git changes, a file, PR, or commit range
- Type `$crit-plus-decide` to turn open questions into a decision page, or supply an existing JSON checklist. The agent reads `crit-plus decide --guide`, waits for your submission, and handles decided, revision-requested, and pending items separately
- In Plan mode with the plugin, proposed plans are reviewed automatically when the agent tries to finish the turn
- The `crit-plus-cli` skill teaches the agent about `crit-plus comment`, sharing, and GitHub PR sync without manual invocation
