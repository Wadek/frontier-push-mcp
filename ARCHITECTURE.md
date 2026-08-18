# Frontier AI-first push — structure this MCP enforces

```
                    FRONTIER AI-FIRST SECURITY
                 (policy first · local first)
  ============================================================

                         REMOTE (git origin)
                            ^
                            | only Executor + fresh gate
                     [ frontier_push ]
                            |
              ┌─────────────▼─────────────┐
              │  1. ROLE CHECK (deny)     │
              │  2. GATE SEAL (ledger)    │
              │  3. LOCAL GIT ONLY        │
              └─────────────┬─────────────┘
                            │
         ┌──────────┬───────┴───────┬──────────┐
         ▼          ▼               ▼          ▼
     OBSERVER   ANALYST        OPERATOR    EXECUTOR
     observe    analyze        prepare     push
     ledger     (egress-safe)  + gate      (no bypass)
                            │
              ┌─────────────▼─────────────┐
              │  LEDGER (.frontier/*.jsonl)│
              │  append-only · hashed     │
              └───────────────────────────┘

  Simple models stay left. Elevate one step at a time.
  ============================================================
```

Maps to Grok-style controls without requiring Grok:

| Idea | Here |
|------|------|
| deny > allow | `policy.Require` |
| no yolo executor | push denied without `gate.passed` |
| evidence | ledger JSONL |
| egress | `analyze` returns summary only |
| sandbox | out of process — run MCP under your OS sandbox |
