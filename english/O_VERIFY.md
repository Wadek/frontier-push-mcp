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
5. PR body = Opt report section + "Verification" (commands + results)
6. Merge; close the issue
```

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
pytest tests/test_api_equivalence.py -q
```

Covers: list/create/toggle/update/delete payloads; static `/` and `/main.js` bytes vs disk (or vs baseline).

### B. Browser (Playwright)

```text
playwright install chromium
pytest tests/test_browser_smoke.py -q
```

Covers: open UI, add task, toggle, edit, delete, search still filters (token optional via env `TASKS_E2E_BASE`).

Default base URL: `http://127.0.0.1:8088` if published, or gateway URL. For compose-on-`waka-net` only, run a side publish or `docker exec` + port-forward — playbook may set `TASKS_E2E_BASE=https://tasks.wakalabs.net?token=…` for real gateway tests (secret via env, never commit).

### Local loop for one Opt

```text
# baseline
pytest tests/ -q
# implement Opt-00N
pytest tests/ -q
frontier guard && frontier plan && frontier apply
# PR with Opt section + Verification
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
