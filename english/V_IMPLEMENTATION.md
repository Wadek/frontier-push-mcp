# How V is implemented (security only)

## Shape

```
  english/SECURITY_POLICY_OWASP.md     what humans mean
           │
  haskell/src/Frontier/OWASP.hs        proof shelf (rule ids in V)
           │
  internal/owasp (Go)                  runs checks on the tree
           │
  git frontier exam | gate | push      control_point = changeset
           │
  ledger                               F0 evidence
```

## Commands you can see

| Command | What it does |
|---------|----------------|
| `git frontier mock-import` | **Mock** of V-importer: skill-table + OWASP rows with control_point + disposition |
| `git frontier exam` | Run current V on this repo; print findings + disposition; seal `exam.owasp` |
| `git frontier gate` | Exam + push rules (branch/dirty/main) + seal pass/fail |
| `git frontier demo` | Ladder + gate preview |

## Dispositions (V only)

| Disposition | When |
|-------------|------|
| **block** | High/Critical under V (blocks gate/push) |
| **advise** | Non-blocking findings (reserved as V grows) |
| **record** | Clean under V, or catalog-only definitions |

## Not built yet (importer real)

Harvest MITRE/CWE/CAPEC into the same columns as `mock-import`, emit Haskell, wire Go only for **changeset**-capable rows.

## Stick with V

Slim/optimize policy is **not** in scope right now.
