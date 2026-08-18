# Frontier Push MCP

**Frontier AI-first security** for teaching models how to push code — **local first**, **simple is better**.

Small models do small steps. Each step is one MCP tool. Evidence is written to a local ledger before anything mutates remotes.

```
  FRONTIER PUSH (local first)
  ---------------------------
  1 HOOKS / POLICY     deny > ask > allow (in-process)
  2 ROLES              Observer → Analyst → Operator → Executor
  3 LEDGER             append-only JSONL (every action)
  4 EGRESS             push metadata only until Executor+gate
  5 OS                 you keep the sandbox; we stay boring

  Cloud models should see status/diffs summaries — not your secrets.
```

## Why this exists

Most agents try to `git push` in one shot. That burns tokens, skips review, and trains models to be reckless.

This MCP forces a **ladder**:

| Role | May do | May not |
|------|--------|---------|
| **Observer** | status, log, ledger tail | write / commit / push |
| **Analyst** | summarize diff, recommend | commit / push |
| **Operator** | branch, stage, commit | push |
| **Executor** | push **only after gate** | skip the ladder |

Simple models stay on Observer/Analyst. Bigger models earn Operator/Executor through explicit `frontier_elevate` — not by default.

## Quick start

```bash
# requires Go 1.22+
git clone https://github.com/Wadek/frontier-push-mcp.git
cd frontier-push-mcp
go build -o frontier-mcp ./cmd/frontier-mcp
go build -o frontier-git ./cmd/frontier-git   # PATH shim over real git

# run MCP against a git repo you want to teach on
export FRONTIER_REPO=/path/to/your/repo
./frontier-mcp
```

### The interface is `git`

Frontier ships a PATH shim also named `git`. You type normal git; high-blast verbs are gated.

```powershell
go build -o D:\frontier\bin\git.exe ./cmd/frontier-git
# FRONTIER_GIT_BIN=C:\Program Files\Git\cmd\git.exe
# D:\frontier\bin first on PATH

git frontier explain
git checkout -b frontier/topic
git add -A; git commit -m "msg"
git frontier gate
git push -u origin HEAD
```

Meta: `git frontier status|gate|ledger|explain`  
Learn: [`teach/LEARN_GIT.md`](teach/LEARN_GIT.md) — kernels later.

### Claude Desktop / Cursor / Grok MCP config (stdio)

```json
{
  "mcpServers": {
    "frontier-push": {
      "command": "/path/to/frontier-mcp",
      "env": {
        "FRONTIER_REPO": "/path/to/repo",
        "FRONTIER_LEDGER": "/path/to/repo/.frontier/ledger.jsonl"
      }
    }
  }
}
```

## Tools (keep them few)

| Tool | Min role | Purpose |
|------|----------|---------|
| `frontier_whoami` | any | role + repo + principles |
| `frontier_observe` | Observer | `git status` / branch / dirty |
| `frontier_analyze` | Analyst | diffstat + short advice |
| `frontier_elevate` | any | step up one role (logged) |
| `frontier_prepare` | Operator | create branch, add, commit |
| `frontier_gate` | Operator | conscience check before push |
| `frontier_push` | Executor | `git push` only if gate passed |
| `frontier_ledger` | Observer | last N ledger events |

## Teaching loop (for model trainers)

Point a **small local model** at this MCP and only allow:

1. `frontier_whoami`
2. `frontier_observe`
3. `frontier_analyze`

It learns to look before writing. Promote to Operator only for commit drills. Executor only on a throwaway remote.

See [`teach/CURRICULUM.md`](teach/CURRICULUM.md).

**Restricted / corporate AI (no install):** copy-paste [`teach/CORPORATE_AI_TRIAL_PROMPT.md`](teach/CORPORATE_AI_TRIAL_PROMPT.md) into Copilot/ChatGPT Enterprise/etc. to simulate the ladder.

## Principles

1. **Local first** — ledger and git on disk; no cloud required.
2. **Simple is better** — stdlib Go, no framework maze.
3. **Simple models do simple things** — one tool ≈ one GPU-cheap step.
4. **Evidence before remote mutate** — gate seal required for push.
5. **No bypass for Executor** — push tool refuses without a fresh gate.

## Axioms

The Frontier AI universe is governed by **F0–F4** (evidence, non-harm, obedience, continuity, examination), plus meta-law **F5 Consilience** (≥3 Turing-complete implementations of any new behavior).  
See [`AXIOMS.md`](AXIOMS.md) and [`META_LANGUAGE.md`](META_LANGUAGE.md) — logic is the universe language; Go is the intermediary runtime, not the sky.

## License

MIT — use it to train, fork, and harden.
