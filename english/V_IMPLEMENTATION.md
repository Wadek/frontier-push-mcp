# How V is implemented (security only)

## Shape

```
  english/SECURITY_POLICY_OWASP.md     what humans mean
           │
  haskell/src/Frontier/OWASP.hs        proof shelf (rule ids in V)
           │
  internal/owasp + internal/vscan      programmatic checks (no tokens)
           │
  git frontier V | gate | push         control_point = changeset
           │
  frontier enhance V                   lean brief → host model (residual only)
           │
  ledger                               F0 evidence
```

## Programmatic first (token thrift)

Do as much as possible **without** an LLM:

| Layer | Tool | Tokens? |
|-------|------|---------|
| Built-in OWASP v0 | `internal/owasp` regex ScanTree | no |
| Adapters | Checkov (if installed); gitleaks/semgrep planned | no |
| Scope + inventory | git diff vs main, lang/manifest counts | no |
| Enhance brief | `.frontier/enhance/V-*.md` capped (~12 KiB) | handoff only |

`enhance V` tells the host model: **do not re-litigate programmatic hits** — only residual SAST/DAST/pentest gaps.

Gate/plan still **hard-block only** on built-in OWASP High/Critical. Adapter + enhance findings default to **advise** until promoted into English→Haskell→Go V.

## Commands you can see

| Command | What it does |
|---------|----------------|
| `git frontier mock-import` | **Mock** of V-importer: skill-table + OWASP rows with control_point + disposition |
| `git frontier V` / `exam` | Built-in OWASP exam; seal `exam.owasp` |
| `git frontier V list` | Built-in + adapters (available?) |
| `git frontier V checkov` | Run Checkov adapter if on PATH |
| `git frontier enhance V` | Programmatic pack + lean brief; seal `enhance.requested` |
| `git frontier enhance seal PATH` | Host result JSON → `enhance.completed` (advise) |
| `git frontier gate` | Exam + push rules (branch/dirty/main) + seal pass/fail |
| `git frontier demo` | Ladder + gate preview |

## Dispositions (V only)

| Disposition | When |
|-------------|------|
| **block** | High/Critical under built-in V (blocks gate/push) |
| **advise** | Non-blocking findings; enhance results; most adapter hits |
| **record** | Clean under V, or catalog-only definitions |

## Not built yet (importer real)

Harvest MITRE/CWE/CAPEC into the same columns as `mock-import`, emit Haskell, wire Go only for **changeset**-capable rows.

## Stick with V

Slim/optimize policy is **not** in scope right now.
