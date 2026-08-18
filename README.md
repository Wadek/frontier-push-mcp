# Frontier Push

Teach AI (and humans) to ship code the safe way.

**The daily interface is `git`.**  
**Humans read simple English.**  
**Proofs live in Haskell.**  
**The local runtime is Go.**  
**Draft in any language — then reverse-engineer to the proof.**  
**Least code that still proves the result wins** (limits vibe bloat and tokens).

```
  English  →  what we mean          (english/)
  Haskell  →  what is true          (haskell/)
  Go       →  what runs             (cmd/, internal/)
  *        →  what you may draft in — then reduce
```

No custom OS. Local first. Simple is better. See the test: `git frontier demo`.

---

## Init (start here)

Full steps: **[english/INIT.md](english/INIT.md)**

Short version:

```powershell
git clone https://github.com/Wadek/frontier-push-mcp.git
cd frontier-push-mcp
powershell -File scripts\install-git-interface.ps1
# open a NEW terminal
git frontier explain
```

First drill:

```powershell
$env:FRONTIER_SOFT = "1"          # learn mode; set to 0 later
git checkout -b frontier/topic
# edit files
git add -A
git commit -m "frontier: topic"
git frontier gate
git push -u origin HEAD
```

---

## What the `git` shim does

| Command | Behavior |
|---------|----------|
| Most `git …` | Passed through to real Git |
| `git commit` on `main`/`master` | Denied (unless `FRONTIER_SOFT=1`) |
| `git push` | Needs feature branch, clean tree, and `git frontier gate` |
| `git frontier status\|gate\|ledger\|explain` | Frontier meta |

Real Git stays at `FRONTIER_GIT_BIN` (default: Git for Windows).

---

## Roles (ladder)

| Role | May | May not |
|------|-----|---------|
| Observer | look | write / push |
| Analyst | summarize | commit / push |
| Operator | branch, commit, gate | push |
| Executor | push after gate | skip the ladder |

Same idea in English (`english/AXIOMS.md`), Haskell (`Frontier.Role`), and Go (`internal/role`).

---

## Repo map

| Path | Layer |
|------|--------|
| [english/INIT.md](english/INIT.md) | How to start |
| [english/LANGUAGE.md](english/LANGUAGE.md) | English · Haskell · Go (+ draft in any) |
| [english/MINIMALITY.md](english/MINIMALITY.md) | Least code; reverse-engineer pushes |
| [english/CONTRIBUTING.md](english/CONTRIBUTING.md) | How to contribute without noise |
| [english/AXIOMS.md](english/AXIOMS.md) | Laws F0–F5 in English |
| [haskell/](haskell/) | Laws as pure Haskell |
| [cmd/frontier-git](cmd/frontier-git) | Go: `git` interface (`git frontier demo`) |
| [cmd/frontier-mcp](cmd/frontier-mcp) | Go: MCP server for AI hosts |
| [teach/](teach/) | Drills and corporate trial prompt |
| [english/WHY_NOT_A_CUSTOM_OS.md](english/WHY_NOT_A_CUSTOM_OS.md) | Why we do not write an OS |

---

## Optional: check Haskell laws

```text
cd haskell
cabal test
```

Needs GHC. Skip until you install it; English + Go still work.

---

## Optional: MCP for AI hosts

Build `frontier-mcp`, then point your host at it (`examples/mcp.claude.json`).  
No install allowed? Use [teach/CORPORATE_AI_TRIAL_PROMPT.md](teach/CORPORATE_AI_TRIAL_PROMPT.md).

---

## License

MIT
