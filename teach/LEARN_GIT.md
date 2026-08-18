# Learn git *through* Frontier (userland first)

You do **not** need to rebuild the Linux kernel to get value.  
First we wrap **git** (the tool everyone already pretends to know). Kernels / “linux frontier” come later.

## What you have locally

| Path | What |
|------|------|
| `D:\frontier\src\git` | Upstream **git/git** source (study copy, shallow clone) |
| `frontier-git` | Our shim: real git + F0–F4 guards on push / main commits |
| System git | `C:\Program Files\Git\cmd\git.exe` (engine under the hood) |

Reading `builtin/push.c` in the source tree teaches how push works.  
Using `frontier-git` teaches **when you are allowed to push**.

## Install the shim (Windows PowerShell)

```powershell
$env:Path = "C:\Users\waka\sdk\go\bin;C:\Program Files\Git\cmd;" + $env:Path
cd C:\Users\waka\src\frontier-push-mcp
go build -o D:\frontier\bin\frontier-git.exe ./cmd/frontier-git

# put Frontier bin FIRST on PATH for this shell
$env:Path = "D:\frontier\bin;" + $env:Path
$env:FRONTIER_GIT_BIN = "C:\Program Files\Git\cmd\git.exe"

# learning mode (warn, don't block) — turn OFF when ready
$env:FRONTIER_SOFT = "1"

frontier-git frontier explain
frontier-git --version   # should show real git version via passthrough
```

Optional: create an alias so typing `git` means Frontier:

```powershell
Set-Alias git frontier-git -Scope Process
```

## Git basics as a Frontier ladder

Think of normal git as three verbs. Frontier adds seals.

```
  git status     ≈  observer   (frontier-git status / frontier status)
  git diff       ≈  analyst
  git commit     ≈  operator   (blocked on main/master)
  git push       ≈  executor   (needs frontier gate)
```

### Drill 1 — observe

```powershell
cd <your-repo>
frontier-git status
frontier-git frontier status
```

### Drill 2 — branch (never commit on main)

```powershell
frontier-git checkout -b frontier/learn-1
# edit a file
frontier-git add -A
frontier-git commit -m "frontier: learning commit"
```

### Drill 3 — gate then push

```powershell
frontier-git frontier gate     # seals gate.passed or fails
# if ok:
$env:FRONTIER_SOFT = "0"       # real deny mode
frontier-git push -u origin HEAD
```

If gate fails, read the reasons (dirty tree, on main, no seal). Fix; re-gate; push.

### Drill 4 — peek at upstream source (optional)

```powershell
cd D:\frontier\src\git
notepad builtin\push.c
# or
findstr /n "push" Documentation\git-push.txt
```

You are learning **two** gits: the C program, and the Frontier *policy* around it.

## Roadmap (so kernels don’t scare you)

```
  NOW     frontier-git wrapper + study clone of git/git
  NEXT    logic/ Datalog core + Python/Rust witnesses (F5)
  LATER   WSL build of your own git binary from source
  MUCH    "linux frontier" — kernel/policy deep dive only after
          userland habits (status→commit→push+gate) are boring
```

**Simple is better:** master the shim before compiling kernels.
