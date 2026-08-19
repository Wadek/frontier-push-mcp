# L — Learn / Landscape

**L** is the third policy family alongside **V** (security) and **S** (slim).

## Purpose

Ingest and classify a **single project** before changing it. L is the first phase of S and of any habitat/architecture review.

```
  L classify  →  map (kind, compose, langs, secret *surfaces*)
  V           →  security exam on known surface
  S           →  slim using L + (later) coverage   [planned]
```

## Commands

| Command | What |
|---------|------|
| `frontier L` / `frontier L classify` | Classify cwd; write `.frontier/learn/L-*`; seal `learn.classified` |
| `frontier L classify PATH` | Classify that project root |
| `frontier L status` | Recent `learn.*` ledger rows |

## Kinds (v0)

`app_compose` · `network_edge` · `media_stack` · `tooling` · `data_volume` · `skills_pack` · `compose_project` · `app_repo` · `unknown`

## Rules

- Read-only: L does not move/delete files.
- Secret **surfaces** only (path names) — never dump secret values into briefs.
- Run **per project** (e.g. each child of `D:\wakalabs`), not on the umbrella alone.
