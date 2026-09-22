# crit+ — Gemini CLI Integration

crit+ (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Drop-in configuration files that teach Gemini CLI to use crit+ for reviewing plans
and code changes, explaining diffs, and collecting structured decisions.

## What's included

| File | Purpose |
|------|---------|
| `skills/crit-plus-cli/SKILL.md` | CLI reference skill — `crit-plus comment`, `crit-plus pull` / `crit-plus push`, `crit-plus share`, review file format |
| `skills/crit-plus-story/SKILL.md` | Story authoring and follow-up review workflow |
| `skills/crit-plus-decide/SKILL.md` | Decision checklists, choices, and revision feedback |
| `commands/crit-plus.toml` | `/crit-plus` slash command that runs the interactive review loop |
| `commands/crit-plus-story.toml` | `/crit-plus-story` slash command for a chaptered diff walkthrough |
| `commands/crit-plus-decide.toml` | `/crit-plus-decide` slash command for structured human decisions |
| `hooks/settings-snippet.json` | Hook that intercepts plan mode exit and runs `crit-plus plan-hook` for inline plan review |
| `hooks/policy.toml` | Auto-allows `exit_plan_mode` without confirmation (browser UI is the sole gate) |

## Install

```bash
crit-plus install gemini
```

This installs all three skills, all three commands, and the policy to your project
directory and merges the hook into `.gemini/settings.json`. For a global install
(available across all projects):

```bash
cd ~ && crit-plus install gemini
```

## Manual setup

If you prefer to install manually:

1. **Skills** — copy to your project or home directory:
   ```
   skills/crit-plus-cli/SKILL.md  → .gemini/skills/crit-plus-cli/SKILL.md
   skills/crit-plus-story/SKILL.md → .gemini/skills/crit-plus-story/SKILL.md
   skills/crit-plus-decide/SKILL.md → .gemini/skills/crit-plus-decide/SKILL.md
   ```
   Global: the same paths under `~/.gemini/skills/`.

2. **Commands** — copy to your project or home directory:
   ```
   commands/crit-plus.toml → .gemini/commands/crit-plus.toml
   commands/crit-plus-story.toml → .gemini/commands/crit-plus-story.toml
   commands/crit-plus-decide.toml → .gemini/commands/crit-plus-decide.toml
   ```
   Global: the same paths under `~/.gemini/commands/`.

3. **Hook** — merge `hooks/settings-snippet.json` into `.gemini/settings.json` (or `~/.gemini/settings.json`). Add the `hooks` key if it doesn't exist, or merge the `BeforeTool` array into the existing `hooks` object.

4. **Policy** — copy to your project or home directory:
   ```
   hooks/policy.toml → .gemini/policies/crit-plus.toml
   ```
   Global: `~/.gemini/policies/crit-plus.toml`

## Usage

Once installed, use `/crit-plus` in Gemini CLI to start a review loop. The agent will:

1. Run `crit-plus` to open your browser for inline commenting
2. Wait for you to click "Finish Review"
3. Read your comments and address each one
4. Loop until you approve with no remaining comments

Use `/crit-plus-story` for a chaptered explanation of a diff followed by review.
Use `/crit-plus-decide` to choose between proposals or request changes. Decision
mode waits for a submitted result; unanswered items remain pending.
