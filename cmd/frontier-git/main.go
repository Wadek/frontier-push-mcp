package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Wadek/frontier-push-mcp/internal/gitx"
	"github.com/Wadek/frontier-push-mcp/internal/ledger"
	"github.com/Wadek/frontier-push-mcp/internal/policy"
)

// frontier-git: drop-in git wrapper. Most commands pass through.
// Frontier security applies to high-blast verbs (push; commit on main).
//
// Env:
//
//	FRONTIER_GIT_BIN   real git executable (optional)
//	FRONTIER_LEDGER    ledger path (optional; else .frontier/ledger.jsonl upward)
//	FRONTIER_SOFT=1    warn instead of deny (learning mode)
//	FRONTIER_STRICT=1  also require gate for `commit` always (optional hardness)

func main() {
	args := os.Args[1:]
	realGit := findRealGit()

	if len(args) == 0 {
		runPassthrough(realGit, args)
		return
	}

	verb := args[0]
	soft := os.Getenv("FRONTIER_SOFT") == "1"
	strict := os.Getenv("FRONTIER_STRICT") == "1"

	switch verb {
	case "push":
		if err := guardPush(soft); err != nil {
			fail(err)
			return
		}
	case "commit":
		if err := guardCommit(args, soft, strict); err != nil {
			fail(err)
			return
		}
	case "frontier":
		// git frontier status|ledger|explain|demo
		handleMeta(args[1:])
		return
	}

	code := runPassthrough(realGit, args)
	os.Exit(code)
}

func guardPush(soft bool) error {
	cwd, _ := os.Getwd()
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return fmt.Errorf("frontier: not a git work tree? %w", err)
	}
	head, _ := repo.RevParseHead()
	por, _ := repo.StatusPorcelain()

	// Always evaluate local gate rules (main/dirty/empty).
	g := policy.EvaluatePushGate(branch, head, por, false)
	if !g.OK {
		msg := fmt.Sprintf("frontier deny push: %s", strings.Join(g.Reasons, "; "))
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			return nil
		}
		return fmt.Errorf("%s\nhint: use a feature branch, commit cleanly, or: frontier-git frontier gate", msg)
	}

	ledPath := findLedger(cwd)
	led, err := ledger.Open(ledPath)
	if err != nil {
		return err
	}
	ok, detail := policy.FreshGateOK(led, branch, head)
	if !ok {
		_, _ = led.Append("frontier-git", "push.denied", map[string]any{
			"branch": branch, "head": head, "reason": detail,
		})
		msg := fmt.Sprintf("frontier deny push: %s", detail)
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			_, _ = led.Append("frontier-git", "push.soft_allow", map[string]any{"branch": branch, "head": head})
			return nil
		}
		return fmt.Errorf("%s\nrun: frontier-git frontier gate\nthen retry push", msg)
	}
	_, _ = led.Append("frontier-git", "push.authorized", map[string]any{
		"branch": branch, "head": head, "gate": detail,
	})
	fmt.Fprintln(os.Stderr, "frontier: push authorized (gate ok)")
	return nil
}

func guardCommit(args []string, soft, strict bool) error {
	cwd, _ := os.Getwd()
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return nil // let real git error
	}
	onMain := strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master")
	if onMain {
		msg := "frontier deny commit on main/master — create a feature branch first (git checkout -b frontier/...)"
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			return nil
		}
		return fmt.Errorf("%s", msg)
	}
	if strict {
		ledPath := findLedger(cwd)
		led, err := ledger.Open(ledPath)
		if err == nil {
			_, _ = led.Append("frontier-git", "commit.attempt", map[string]any{
				"branch": branch, "args": strings.Join(args, " "),
			})
		}
	}
	return nil
}

func handleMeta(args []string) {
	if len(args) == 0 {
		fmt.Println(`git frontier commands:
  git frontier status    tree + ledger path
  git frontier gate      seal gate for current HEAD
  git frontier ledger    last ledger rows
  git frontier demo      SEE the ladder (visible test)
  git frontier explain   how this wraps git

Env: FRONTIER_SOFT=1 (learn), FRONTIER_STRICT=1, FRONTIER_GIT_BIN, FRONTIER_LEDGER`)
		return
	}
	cwd, _ := os.Getwd()
	switch args[0] {
	case "status":
		repo := gitx.Repo{Dir: cwd}
		b, _ := repo.Branch()
		p, _ := repo.StatusPorcelain()
		fmt.Printf("cwd: %s\nbranch: %s\ndirty: %v\nledger: %s\nreal_git: %s\nsoft: %v\n",
			cwd, b, policy.DirtyPorcelain(p), findLedger(cwd), findRealGit(), os.Getenv("FRONTIER_SOFT") == "1")
	case "gate":
		repo := gitx.Repo{Dir: cwd}
		b, _ := repo.Branch()
		h, _ := repo.RevParseHead()
		p, _ := repo.StatusPorcelain()
		g := policy.EvaluatePushGate(b, h, p, false)
		led, err := ledger.Open(findLedger(cwd))
		if err != nil {
			fail(err)
			return
		}
		sealed, err := policy.SealGate(led, "frontier-git", g)
		if err != nil {
			fail(err)
			return
		}
		fmt.Printf("ok=%v seal=%s reasons=%v\nbranch=%s head=%s\n", sealed.OK, sealed.SealHash, sealed.Reasons, sealed.Branch, sealed.Head)
		if !sealed.OK {
			os.Exit(2)
		}
	case "ledger":
		led, err := ledger.Open(findLedger(cwd))
		if err != nil {
			fail(err)
			return
		}
		rows, _ := led.Tail(15)
		for _, r := range rows {
			fmt.Printf("%d %s %s %s\n", r.Seq, r.TS, r.Actor, r.Action)
		}
	case "demo":
		printDemo(cwd)
	case "explain":
		fmt.Println(`You are talking to Frontier through the git interface.

  Type:  git …              (this shim)
  Engine: FRONTIER_GIT_BIN  (real git under the hood)

Passthrough: almost all git commands.
Guarded: git push   — needs feature branch, clean tree, fresh gate
         git commit — denied on main/master unless FRONTIER_SOFT=1

Meta:    git frontier status|gate|ledger|demo|explain
Languages: English (meaning) · Haskell (proof) · Go (runtime)
Contribute in any language, then reverse-engineer to Haskell proof.
Minimality: least code that still proves the result.
Learn:   set FRONTIER_SOFT=1 then turn it off when ready`)
	default:
		fmt.Fprintf(os.Stderr, "unknown frontier subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func printDemo(cwd string) {
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	p, _ := repo.StatusPorcelain()
	dirty := policy.DirtyPorcelain(p)
	g := policy.EvaluatePushGate(b, h, p, false)
	ledPath := findLedger(cwd)
	var lastGate string
	if led, err := ledger.Open(ledPath); err == nil {
		if e, _ := led.LastAction("gate.passed"); e != nil {
			lastGate = "gate.passed@" + e.EntryHash[:12]
		} else if e, _ := led.LastAction("gate.failed"); e != nil {
			lastGate = "gate.failed"
		} else {
			lastGate = "(none yet)"
		}
	} else {
		lastGate = "(no ledger)"
	}

	onMain := strings.EqualFold(b, "main") || strings.EqualFold(b, "master")
	fmt.Println(`╔══════════════════════════════════════════════╗
║           FRONTIER  —  visible test          ║
╚══════════════════════════════════════════════╝`)
	fmt.Printf("  branch     %s\n", nz(b, "(none)"))
	fmt.Printf("  HEAD       %s\n", short(h))
	fmt.Printf("  dirty      %v\n", dirty)
	fmt.Printf("  on_main    %v\n", onMain)
	fmt.Printf("  gate_now   ok=%v  %v\n", g.OK, g.Reasons)
	fmt.Printf("  last_seal  %s\n", lastGate)
	fmt.Printf("  ledger     %s\n", ledPath)
	fmt.Println()
	fmt.Println("  ladder     Observer → Analyst → Operator → Executor")
	fmt.Println("  push?      only Executor + fresh gate.passed + feature branch")
	fmt.Println("  languages  English · Haskell · Go   (draft in any, prove in Haskell)")
	fmt.Println("  minimality least code that still proves the result")
	fmt.Println()
	if g.OK {
		fmt.Println("  SEE: gate would PASS right now. Next: git push (if role allows).")
	} else {
		fmt.Println("  SEE: gate would FAIL right now. Fix reasons, then: git frontier gate")
	}
	fmt.Println("╚══════════════════════════════════════════════╝")
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	if h == "" {
		return "(none)"
	}
	return h
}

func nz(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func findLedger(cwd string) string {
	if v := os.Getenv("FRONTIER_LEDGER"); v != "" {
		return v
	}
	// Prefer in-repo ledger only if it already exists (user opted in).
	dir := cwd
	for {
		cand := filepath.Join(dir, ".frontier", "ledger.jsonl")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Default: ledger OUTSIDE the work tree so evidence never dirties the diff.
	// (Local-first, still on disk — not cloud.)
	sum := sha256Short(cwd)
	root := os.Getenv("FRONTIER_HOME")
	if root == "" {
		root = filepath.Join("D:\\frontier", "ledgers")
	}
	return filepath.Join(root, sum, "ledger.jsonl")
}

func sha256Short(s string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(s))))
	return hex.EncodeToString(sum[:8])
}

func findRealGit() string {
	if v := os.Getenv("FRONTIER_GIT_BIN"); v != "" {
		return v
	}
	// Prefer system Git for Windows, not ourselves.
	candidates := []string{
		`C:\Program Files\Git\cmd\git.exe`,
		`C:\Program Files\Git\bin\git.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("git"); err == nil {
		// Avoid infinite recursion if frontier-git is named git on PATH
		if abs, err2 := filepath.Abs(os.Args[0]); err2 == nil {
			if filepath.Clean(p) == filepath.Clean(abs) {
				fmt.Fprintln(os.Stderr, "frontier: set FRONTIER_GIT_BIN to real git")
				os.Exit(127)
			}
		}
		return p
	}
	fmt.Fprintln(os.Stderr, "frontier: real git not found")
	os.Exit(127)
	return ""
}

func runPassthrough(git string, args []string) int {
	cmd := exec.Command(git, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
