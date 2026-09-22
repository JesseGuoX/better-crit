# Crit Plus Story — chaptered diff overview

Crit Plus (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Use this workflow only when the user explicitly invokes `/crit-plus-story` or
directly asks you to generate a crit-plus story. Do not infer it from a normal
`/crit-plus` review, PR review, or generic "review this" request.

Primary path: author in-session with `--guide` / `--prep` / `--story-file`
(do **not** run bare `crit-plus story`, which spends `agent_cmd` tokens).

## Step 1: Guide

```bash
crit-plus story --guide
```

Follow the printed guide and JSON schema exactly.

## Step 2: Prep

```bash
crit-plus story --prep /tmp/crit-plus-story-prep.txt
```

Read that file — every hunk id is `(file_path, old_start)`.

## Step 3: Author JSON

Write only `prologue`, `chapters`, and `support` to `/tmp/crit-plus-story.json`.

## Step 4: Ingest

```bash
crit-plus story --story-file /tmp/crit-plus-story.json
```

Fix coverage failures and retry. On drift, re-prep and re-author.

## Step 5: Reconnect and wait for Finish Review

**CRITICAL.** Ingest opens the UI and exits. Run bare `crit-plus` and wait until it
exits (same wait mechanics as the `/crit-plus` / `crit-plus` skill for this agent). Do
not proceed until Finish Review.

```bash
crit-plus
```

## Steps 6–8: Review cycle

Same loop as `/crit-plus`: read finish stdout → address comments on **source files**
(not the story JSON) with `crit-plus comment --reply-to` → run `crit-plus` again for the
next round → stop when approved.

## Out of scope during authoring

- Do not produce agent-authored review comments while writing the story.
- Do not run bare `crit-plus story` (LLM via `agent_cmd`) unless the user asks.
- After ingest, enter the normal review cycle (Steps 5–8); do not edit the
  saved story JSON unless the user explicitly asks.
