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

## Terraform-like flow

```powershell
git checkout -b frontier/topic
# edit…
git add -A
git commit -m "msg"

git frontier plan     # preview V — FAIL stops everything
git frontier apply    # seal authorization — only if plan passed
git push              # only if apply sealed gate.passed
```

| Command | Like Terraform | What it does |
|---------|----------------|--------------|
| `git frontier plan` | `terraform plan` | Run **V**; preview ship; **exit ≠ 0** if block |
| `git frontier apply` | `terraform apply` | Authorize push **only** after fresh `plan.passed` |
| `git push` | apply to remote | Real git; Frontier denies without sealed apply |
| Ledger | state | Evidence of plan/apply (F0) |

Also:

```text
git frontier V              # vulnerabilities exam (alias: exam)
git frontier S              # slim stub — planned, not enforced
git frontier mock-import    # mock V-importer list
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
