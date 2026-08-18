# Frontier Push

Teach AI (and humans) to ship code safely, and **manage vibe code** under policy.

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

| | Name | Meaning |
|--|------|---------|
| **V** | **Vulnerabilities** | Security definitions (OWASP / MITRE via importer). Examined at **changeset**. High/Critical → **block**. |
| **S** | **Slim** | **Planned:** reduce vibe-code bloat (budgets, dead code). Advise first; optional block later. **Not enforced yet.** |

Today: **V only.** S is on the roadmap.

---

## Init

Full steps: **[english/INIT.md](english/INIT.md)**

```powershell
git clone https://github.com/Wadek/frontier-push-mcp.git
cd frontier-push-mcp
powershell -File scripts\install-git-interface.ps1
# new terminal
git frontier explain
```

---

## Two ways to run the same tool

| How | Example |
|-----|---------|
| **With git** (daily) | `git frontier plan` |
| **Standalone** | `frontier plan` or `frontier V` |

Same engine. Not `go frontier` — `go` is the Go *language toolchain* (`go build`, `go test`).  
During development you *may* run `go run ./cmd/frontier V` (that means “run this package,” not a Go subcommand).

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
frontier V                 # vulnerabilities exam (built-in OWASP)
frontier V list            # scanners: owasp-v0, checkov, …
frontier V checkov         # Checkov adapter if installed (no tokens)
frontier enhance V         # programmatic pack + lean brief for host model
frontier S                 # slim stub — planned
frontier mock-import       # mock V-importer
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
