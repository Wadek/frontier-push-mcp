# Optimize verification — prove behavior is unchanged

Optimize (O) only ships when we can show the **end result is the same** for the system and for the **end user**.

This is the standard process to bake into Frontier Ship. Each **code stack** adds a short playbook (how to run tests in that ecosystem).

---

## Standard process (every Opt-ID)

```text
1. Pick ONE open Opt-* issue (small PR)
2. Capture baseline
   a. Equivalence / code tests (API, pure functions, golden outputs)
   b. Browser smoke (critical user paths)
3. Implement the Opt change (simplest behavior-preserving transform)
4. Re-run the same tests — must stay green / same assertions
5. PR body = Opt report section + "Verification" (commands + **paste/verbose log**)
6. Merge; close the issue
```

### Visibility (required)

Customers (and agents) must **see** the proof, not only hear “tests passed.”

| Where | What |
|-------|------|
| **Local / agent terminal** | Always run with **verbose** output, e.g. `pytest -v --tb=short`. Paste the summary (and failures) into the chat/PR. |
| **GitHub (when remote is GitHub)** | Add a **Actions** workflow that runs the same suite on every PR/push. The check must be visible on the PR. |

Do **not** treat a silent `pytest -q` with exit 0 as sufficient communication. Quiet mode is fine for CI logs after the first green run, but humans reviewing Opt work should get a visible list of passed tests.

### Two proof layers

| Layer | Question | Typical tools |
|-------|----------|----------------|
| **A. Equivalence (code)** | Same outputs / contracts for the same inputs? | unit/integration, API clients, golden files |
| **B. Browser (user)** | Same UX for the human path? | **Playwright** (default for web UIs) |

Layer A does **not** replace Layer B. Fast APIs can still break clicks, focus, tokens, empty states.

### What “same” means

- **API / data:** same status codes, JSON fields, ordering guarantees we document, file schema (whitespace-only JSON diffs may be OK if stated).
- **User:** same visible flows (list, add, edit, toggle, delete, search, auth token still works).
- **Not required:** identical timings, log lines, or internal structure.

### Frontier Ship hooks (future)

| Step | Command / artifact (planned) |
|------|------------------------------|
| Record baseline | `frontier optimize verify baseline` |
| After change | `frontier optimize verify` |
| Stack playbook | `.frontier/optimize/stack.md` or `english/stacks/<id>.md` |
| PR checklist | Auto-append Verification section to `optimize pr-body` |

Until those exist: follow this doc manually and paste results into the PR.

---

## Stack playbook — Python + static web UI (tasks)

**Example customer:** `github.com/Wadek/tasks` under `D:\wakalabs\tasks`.

### A. Equivalence

```text
pip install -r requirements-dev.txt
pytest tests/test_api_equivalence.py -v --tb=short
```

Covers: list/create/toggle/update/delete payloads; static `/` and `/main.js` bytes vs disk (or vs baseline).

### B. Browser (Playwright)

```text
playwright install chromium
pytest tests/test_browser_smoke.py -v --tb=short
```

Covers: open UI, add task, toggle, edit, delete, search still filters (token optional via env `TASKS_E2E_BASE`).

Default: ephemeral server from `conftest.py`. Optional gateway: `TASKS_E2E_BASE=https://tasks.wakalabs.net?token=…` (secret via env, never commit).

### GitHub Actions (customer has GitHub)

Workflow: `.github/workflows/verify.yml` — runs `pytest -v` (API + Playwright) on PR/push to `main`.  
That is the customer-visible proof on the PR checks panel.

### Local loop for one Opt

```text
# baseline (verbose — show in terminal)
pytest -v --tb=short
# implement Opt-00N
pytest -v --tb=short
frontier guard && frontier plan && frontier apply
# PR with Opt section + Verification + CI green
```

---

## Stack playbooks to add later

| Stack | Equivalence | Browser |
|-------|-------------|---------|
| Node/Express | jest/vitest + supertest | Playwright |
| Go HTTP | `go test` + httptest | Playwright |
| Pure library (no UI) | unit only | N/A — skip Layer B with note |

---

## Relation to Learn / Guard / Slim

- **Learn:** know the project kind (web? API-only?).
- **Guard:** security still clean after Opt.
- **Slim:** do not confuse deletion of unused code with Optimize.
- **Optimize:** speed with **A + B** proofs.
