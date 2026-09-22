# crit+ for Codex

Review code and Markdown in your browser, leave inline feedback, and make
structured decisions with your agent.

**crit+ is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit),
originally created by Tomasz Tomczyk.** This fork is maintained by JesseGuoX;
the upstream MIT license and copyright are preserved. See
[LICENSE](https://github.com/JesseGuoX/crit-plus/blob/main/LICENSE) and
[NOTICE](https://github.com/JesseGuoX/crit-plus/blob/main/NOTICE).

## What you can do

- Review git changes, plans, Markdown files, live pages, and local HTML with inline comments and follow-up rounds.
- Read chaptered stories that explain a diff before reviewing it.
- Compare decision items, options, and recommendations; select choices or request changes. Partial submissions keep unanswered items pending.
- Use the refreshed Markdown layout on laptop and wide-screen displays.
- Review in-chat proposed plans through the Codex plan-review hook.

## Install

Install the [crit-plus CLI](https://github.com/JesseGuoX/crit-plus#1-build-and-install-crit-plus)
and make sure `crit-plus --version` works in your terminal. Then run:

```bash
crit-plus install codex-plugin          # current project
# Or install for all projects:
cd ~ && crit-plus install codex-plugin
```

This registers the plugin in a local marketplace, enables `crit-plus@local`,
and installs loose skills so the short skill names below work. Start a new Codex
session to load the integration. Use `--force` to refresh existing loose skills;
plugin files and the activation cache are refreshed on every plugin install.

The marketplace display name is **crit+**. The plugin ID, executable, and skill
names use `crit-plus`.

## Try it

- `$crit-plus` — review current changes, a file, a PR, or a page.
- `$crit-plus-story` — create a chaptered diff walkthrough and continue the review.
- `$crit-plus-decide` — turn open questions into a decision page with options and recommendations.
- `crit-plus-cli` — teaches the agent how to read review results, reply to comments, share reviews, and sync PR feedback.

Decision checklists use JSON, with Markdown supported in context and description
fields. Your content and feedback keep their original language; built-in UI text
is English. The agent waits for your submission before applying your choices.

For skills without plugin hooks, run `crit-plus install codex`. To disable
automatic plan review, set `CRIT_PLAN_REVIEW=off`.

[Project and CLI documentation](https://github.com/JesseGuoX/crit-plus) ·
[Codex installation and hooks](https://github.com/JesseGuoX/crit-plus/blob/main/integrations/codex/README.md)
