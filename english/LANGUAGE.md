# Languages of Frontier

We do not invent random human slang for machines.

| Layer | Who it is for | Language |
|-------|----------------|----------|
| **Human** | People reading and teaching | **Simple English** |
| **Compute / AI** | Laws, roles, gates as pure logic | **Haskell** (for now) |
| **Runtime** | Local process, git, MCP, disk | **Go** |

```
  English  →  what we mean
  Haskell  →  what is true (types + pure functions)
  Go       →  what runs on your machine
```

## Rule F5 (this repo)

A new Frontier behavior is done when:

1. It is described in **English** under `english/`, and  
2. It is expressed in **Haskell** under `haskell/`, and  
3. It is implemented in **Go** under `cmd/` / `internal/`.

Same meaning in all three. No fourth “whatever language we felt like today.”

Python, Ruby, ad-hoc shell, and blog dialects are **not** Frontier languages here.
