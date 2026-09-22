# Crit Plus — Gemini CLI Integration

Crit Plus (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Drop-in configuration files that teach Gemini CLI to use Crit Plus for reviewing plans and code changes.

## What's included

| File | Purpose |
|------|---------|
| `skills/crit-plus-cli/SKILL.md` | CLI reference skill — `crit-plus comment`, `crit-plus pull/push`, `crit-plus share`, review file format |
| `commands/crit-plus.toml` | `/crit-plus` slash command that runs the interactive review loop |
| `hooks/settings-snippet.json` | Hook that intercepts plan mode exit and runs `crit-plus plan-hook` for inline plan review |
| `hooks/policy.toml` | Auto-allows `exit_plan_mode` without confirmation (browser UI is the sole gate) |

## Install

```bash
crit-plus install gemini
```

This installs the skill, command, and policy to your project directory and merges the hook into `.gemini/settings.json`. For a global install (available across all projects):

```bash
cd ~ && crit-plus install gemini
```

## Manual setup

If you prefer to install manually:

1. **Skill** — copy to your project or home directory:
   ```
   skills/crit-plus-cli/SKILL.md  → .gemini/skills/crit-plus-cli/SKILL.md
   ```
   Global: `~/.gemini/skills/crit-plus-cli/SKILL.md`

2. **Command** — copy to your project or home directory:
   ```
   commands/crit-plus.toml → .gemini/commands/crit-plus.toml
   ```
   Global: `~/.gemini/commands/crit-plus.toml`

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
