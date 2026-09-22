# crit+ for Cursor

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

## Install

Install the [crit-plus CLI](https://github.com/JesseGuoX/crit-plus#1-build-and-install-crit-plus)
and make sure `crit-plus --version` works in your terminal. For a project-local
installation, run:

```bash
crit-plus install cursor
```

This installs four skills under `.cursor/skills/`. Run
`crit-plus install cursor --force` to refresh existing skills.

For marketplace distribution, the repository's `.cursor-plugin/marketplace.json`
points to this directory, which includes `.cursor-plugin/plugin.json` and the
same four skills. The plugin ID and executable are `crit-plus`; the brand is
**crit+**.

## Try it

- `/crit-plus` — review current changes, a file, a PR, or a page.
- `/crit-plus-story` — create a chaptered diff walkthrough and continue the review.
- `/crit-plus-decide` — turn open questions into a decision page with options and recommendations.
- `crit-plus-cli` — teaches the agent how to read review results, reply to comments, share reviews, and sync PR feedback.

Decision checklists use JSON, with Markdown supported in context and description
fields. Your content and feedback keep their original language; built-in UI text
is English. The agent waits for your submission before applying your choices.

[Project and CLI documentation](https://github.com/JesseGuoX/crit-plus) ·
[All integrations](https://github.com/JesseGuoX/crit-plus/blob/main/integrations/README.md)
