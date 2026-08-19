# Frontier Ship

Manage code in a frontier-AI landscape — **ship safely** (V) and **stay slim** (S).

**The daily interface is `git`.**  
**MCP** connects hosts and (later) agents.  
**Humans keep control at push.**

```
  English  →  what we mean
  Haskell  →  what is true
  Go       →  what runs
```

Local first. Simple is better. Fail closed — like Terraform: **nothing goes if plan/apply fails.**  
**We dogfood:** this repo ships through `frontier plan → apply → push` ([english/DOGFOOD.md](english/DOGFOOD.md)).

---

## Policy families

| Letter | Word | Meaning |
|--------|------|---------|
| **L** | **Learn** | Ingest + classify a project before change (first phase of Slim). |
| **G** | **Guard** | Security (OWASP / secret surfaces / adapters). Examined at **changeset**. High/Critical → **block**. |
| **S** | **Slim** | **Planned:** reduce vibe-code bloat. Advise first; optional block later. **Not enforced yet.** |

Today: **Learn + Guard.** Slim is on the roadmap. Prefer full words in scripts (`frontier guard`); letters are aliases (`G`).

---

## Init

Full steps: **[english/INIT.md](english/INIT.md)**

```powershell
git clone https://github.com/Wadek/frontier-ship.git
cd frontier-ship
powershell -File scripts\install-git-interface.ps1
# new terminal
git frontier explain
```

---

## Two ways to run the same tool

| How | Example |
|-----|---------|
| **With git** (daily) | `git frontier plan` |
| **Standalone** | `frontier plan` or `frontier guard` |

Same engine. Not `go frontier` — `go` is the Go *language toolchain* (`go build`, `go test`).  
During development you *may* run `go run ./cmd/frontier guard` (that means “run this package,” not a Go subcommand).

---

## Terraform-like flow

```powershell
git checkout -b frontier/topic
# edit…
git add -A
git commit -m "msg"

git frontier plan     # or: frontier plan
git frontier apply    # or: frontier apply
git push
```

| Command | Like Terraform | What it does |
|---------|----------------|--------------|
| `frontier plan` / `git frontier plan` | `terraform plan` | Run **V**; preview; **exit ≠ 0** if block |
| `frontier apply` / `git frontier apply` | `terraform apply` | Authorize push only after `plan.passed` |
| `git push` | mutate remote | Denied without sealed apply |
| Ledger | state | Evidence of plan/apply (F0) |

Also:

```text
frontier learn             # L — classify this project
frontier guard             # G — security exam + secret surfaces
frontier guard list        # scanners: owasp-v0, checkov, …
frontier enhance guard     # programmatic pack + lean brief for host model
frontier slim              # S — planned
frontier mock-import       # mock importer
git frontier demo|ledger|status|explain
```
---

## Repo map

| Path | What |
|------|------|
| [english/INIT.md](english/INIT.md) | Start here |
| [english/V_IMPLEMENTATION.md](english/V_IMPLEMENTATION.md) | How V works |
| [english/SECURITY_POLICY_OWASP.md](english/SECURITY_POLICY_OWASP.md) | V policy (OWASP) |
| [english/CUSTOMER_JOURNEY.md](english/CUSTOMER_JOURNEY.md) | Takeover / vibe-code use |
| [english/SCORING_AND_BENCHMARKS.md](english/SCORING_AND_BENCHMARKS.md) | Official score meaning |
| [haskell/](haskell/) | Proof form of laws + OWASP |
| [cmd/frontier-git](cmd/frontier-git) | `git` interface |

---

## Alpha release

[english/RELEASE.md](english/RELEASE.md) — SLSA Go releaser, tag `v0.1.0-alpha.1`.

## License

MIT
