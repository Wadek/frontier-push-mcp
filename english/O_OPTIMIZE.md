# Optimize (O) — behavior-preserving speed

**Word:** `frontier optimize`  
**Letter alias:** `O` (prefer the word in scripts)

Optimize runs **after** Learn → Guard → Slim. It applies theoretical CS judgment at the code level to find the **simplest** change that **does not alter intended behavior**.

## Pipeline order

```
  frontier scm …     # if customer has no VCS (separate command)
  frontier learn     # L — landscape
  frontier guard     # G — security
  frontier slim      # S — bloat (planned)
  frontier optimize  # O — speed (this family)
```

Axioms apply throughout (F0 evidence, fail-closed ship).

## What it is

Report-driven proposals such as:

- Asymptotic waste (when an equivalent cheaper structure exists)
- Redundant work / N+1 when results stay identical
- Hot-path vs cold-path separation
- Data-structure fit under unchanged equality/API contracts
- I/O and allocation thrash when outputs stay equivalent

## What it is not

Feature work, UX changes, speculative micro-opts without a reason, full rewrites, or LLM-only vibe edits.

## Developer report (canonical in the PR)

Each finding is one review unit (**one small PR** preferred):

```text
### Opt-001 — <title>
- Location: path  function  lines
- Intended behavior: …
- Why wasteful: … (plain English + O(…) if useful)
- Suggested change: …
- Snippet before / after
- Risk / equivalence / tests
- Disposition: advise (default)
```

| Place | Role |
|-------|------|
| **PR body** | Canonical report humans read |
| Commit message | References Opt-ID |
| `.frontier/optimize/` (later) | Offline copy + ledger |
| Diff | The change |

## Small PRs

Never one mega Optimize PR. One Opt-ID (or tight cluster) per branch: `frontier/opt-<id>-…`.

## Status

**Phase A:** doctrine + CLI stub.  
Detectors and report writers come later. Enhance path: `frontier enhance optimize` (planned).
