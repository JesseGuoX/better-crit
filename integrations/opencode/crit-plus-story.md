---
description: Author a crit-plus story (chaptered diff overview) — only when explicitly invoked
agent: build
---

# Author a crit-plus story with `crit-plus story`

Crit Plus (`crit-plus`) is an enhanced fork of [Crit](https://github.com/tomasz-tomczyk/crit), originally created by **Tomasz Tomczyk**. The upstream MIT license and copyright are preserved.

Invoke this only when the user runs `/crit-plus-story` or directly asks you to
generate a crit-plus story. Do not infer it from generic review, PR, `/crit-plus`, or
diff-review requests.

Primary path: author in-session with `--guide` / `--prep` / `--story-file`
(do **not** run bare `crit-plus story`, which spends `agent_cmd` tokens).

## Step 1: Fetch the guide

```bash
crit-plus story --guide
```

Read and follow that guide's principles and JSON shape exactly.

## Step 2: Write the prep file

```bash
crit-plus story --prep /tmp/crit-plus-story-prep.txt
```

**Read that file** — the diff is never inlined into the guide prompt.

## Step 3: Author the story JSON

Cluster hunks by theme (not by file). Write JSON with **only** `prologue`,
`chapters`, and `support` to `/tmp/crit-plus-story.json`. Crit Plus fills metadata.

## Step 4: Ingest

```bash
crit-plus story --story-file /tmp/crit-plus-story.json
```

Exit 0 = saved (browser opens). Exit 1 = rejected — fix coverage and retry.
On drift, re-run `--prep` and re-author.

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
