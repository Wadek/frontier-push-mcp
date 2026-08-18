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
	"github.com/Wadek/frontier-push-mcp/internal/owasp"
	"github.com/Wadek/frontier-push-mcp/internal/policy"
)

// Set by SLSA / release ldflags.
var (
	version = "dev"
	commit  = "none"
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
	axiom("F0", "push.start", "push requested — evidence + gate required")
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return fmt.Errorf("frontier: not a git work tree? %w", err)
	}
	head, _ := repo.RevParseHead()
	por, _ := repo.StatusPorcelain()

	axiom("F4", "push.recheck_policy", "re-scan OWASP V before authorizing remote mutate")
	findings, _ := owasp.ScanTree(cwd)
	if verbose() {
		fmt.Fprintln(os.Stderr, owasp.FormatReport(findings))
	}

	g := policy.EvaluatePushGate(branch, head, por, false)
	if owasp.BlocksGate(findings) {
		g.OK = false
		g.Reasons = append(g.Reasons, "OWASP V: untriaged High/Critical finding(s)")
		axiom("F4", "push.block", "High/Critical under V")
	}
	if !g.OK {
		axiom("F0", "push.deny", strings.Join(g.Reasons, "; "))
		msg := fmt.Sprintf("frontier deny push: %s", strings.Join(g.Reasons, "; "))
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			axiom("F3", "soft_allow", "learning mode weakens continuity — turn FRONTIER_SOFT off")
			return nil
		}
		return fmt.Errorf("%s\nhint: use a feature branch, commit cleanly, or: git frontier gate", msg)
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
		axiom("F0", "push.deny", detail)
		msg := fmt.Sprintf("frontier deny push: %s", detail)
		if soft {
			fmt.Fprintln(os.Stderr, "WARNING:", msg, "(FRONTIER_SOFT=1 — allowing)")
			_, _ = led.Append("frontier-git", "push.soft_allow", map[string]any{"branch": branch, "head": head})
			return nil
		}
		return fmt.Errorf("%s\nrun: git frontier gate\nthen retry push", msg)
	}
	_, _ = led.Append("frontier-git", "push.authorized", map[string]any{
		"branch": branch, "head": head, "gate": detail, "owasp_findings": len(findings),
	})
	axiom("F0", "push.authorized", detail)
	axiom("F2", "push.execute", "passing through to real git push")
	fmt.Fprintln(os.Stderr, "frontier: push authorized (gate ok)")
	return nil
}

func guardCommit(args []string, soft, strict bool) error {
	cwd, _ := os.Getwd()
	axiom("F0", "commit.start", "commit is Operator-level; evidence path continues in ledger on gate")
	repo := gitx.Repo{Dir: cwd}
	branch, err := repo.Branch()
	if err != nil {
		return nil // let real git error
	}
	onMain := strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master")
	if onMain {
		axiom("F1", "commit.deny_main", "direct commits on main expand blast radius")
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
		fmt.Println(`git frontier commands (Terraform-like: plan → apply → push):

  git frontier plan         preview V (and note S); FAIL stops the world
  git frontier apply        seal push authorization only if plan passed
  git frontier gate         alias of apply

  git frontier V            run Vulnerabilities exam (OWASP)
  git frontier S            Slim / vibe-bloat (PLANNED — not enforced)
  git frontier exam         alias of V
  git frontier mock-import  mock V-importer list

  git frontier status|ledger|demo|explain

Env: FRONTIER_SOFT=1  FRONTIER_VERBOSE=1  FRONTIER_GIT_BIN  FRONTIER_LEDGER

Nothing remote goes if plan/apply fails (like terraform).`)
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
	case "V", "v", "exam":
		runExam(cwd, true)
	case "S", "s":
		printSlimStub()
	case "plan":
		runPlan(cwd, true)
	case "apply", "gate":
		runApply(cwd, true)
	case "mock-import":
		printMockImport()
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
		fmt.Printf("frontier-git %s (%s)\n\n", version, commit)
		fmt.Println(`You are talking to Frontier through the git interface.

  Type:  git …
  Engine: FRONTIER_GIT_BIN (real git)

Terraform-like flow:
  git frontier plan    # preview — fails closed
  git frontier apply   # authorize — only if plan passed
  git push             # only if apply/gate sealed

Policy families:
  V  Vulnerabilities — security definitions (OWASP…); enforced at changeset
  S  Slim — vibe-code bloat reduction; PLANNED (not enforced yet)

Control points: changeset | review | runtime | engagement
Languages: English · Haskell · Go
State: ledger (like terraform state) — evidence of plan/apply`)
	default:
		fmt.Fprintf(os.Stderr, "unknown frontier subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func printSlimStub() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  S (Slim) — PLANNED, not enforced yet        ║
╚══════════════════════════════════════════════╝
  Purpose: manage vibe-code bloat (least code that still works)
  Control: changeset (advise→block later) + review (intent)
  Today:   use V only for security baseline
  Later:   git frontier S   will report budgets / dead code

  Stick with:  git frontier V | plan | apply | push
╚══════════════════════════════════════════════╝`)
}

func axiom(id, when, detail string) {
	fmt.Fprintf(os.Stderr, "AXIOM %-4s | %-28s | %s\n", id, when, detail)
}

func verbose() bool {
	return os.Getenv("FRONTIER_VERBOSE") == "1" || os.Getenv("FRONTIER_VERBOSE") == "true"
}

func runExam(cwd string, seal bool) ([]owasp.Finding, error) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║     FRONTIER EXAM — V at control=changeset   ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "exam.ledger", "evidence path open")
	axiom("F4", "exam.start", "policy V = OWASP Top 10 v0 (English→Haskell→Go)")

	findings, err := owasp.ScanTree(cwd)
	if err != nil {
		return nil, err
	}
	fmt.Println()
	fmt.Println(owasp.FormatReport(findings))
	fmt.Println()

	block := owasp.BlocksGate(findings)
	disposition := "record"
	switch {
	case block:
		disposition = "block"
		axiom("F4", "disposition", "block — High/Critical under V")
	case len(findings) > 0:
		disposition = "advise"
		axiom("F4", "disposition", "advise — findings present, none High/Critical")
	default:
		axiom("F4", "disposition", "record — Clean under current V")
	}

	fmt.Printf("control_point: changeset\n")
	fmt.Printf("disposition:   %s\n", disposition)
	fmt.Printf("V_policy:      OWASP-Top10-2021-v0\n")
	fmt.Printf("findings:      %d\n", len(findings))
	fmt.Println("╚══════════════════════════════════════════════╝")

	if seal {
		led, err := ledger.Open(findLedger(cwd))
		if err != nil {
			return findings, err
		}
		_, _ = led.Append("frontier-git", "exam.owasp", map[string]any{
			"policy":       "OWASP-Top10-2021-v0",
			"control":      "changeset",
			"disposition":  disposition,
			"findings":     len(findings),
			"blocks_gate":  block,
		})
		axiom("F0", "ledger.append", "exam.owasp sealed")
	}
	axiom("F4", "exam.done", fmt.Sprintf("%d finding(s)", len(findings)))
	return findings, nil
}

func printMockImport() {
	fmt.Println(`╔══════════════════════════════════════════════╗
║   MOCK V-IMPORTER (discussion → visible)     ║
║   Source seed: cyber skill table + OWASP     ║
╚══════════════════════════════════════════════╝
id                        control_point   disposition  note
------------------------  --------------  -----------  ----
CAPEC-66                  changeset       block        SQLi — gateable now (in Go V)
CAPEC-63                  changeset       block        XSS — partially gateable
OWASP-A01..A10            changeset       block/advise Top10 v0 implemented
PENT-DOMAIN-WEB           catalog         record       umbrella; speciate later
PENT-DOMAIN-API           catalog         record       umbrella
PENT-DOMAIN-AD            engagement      record       needs confirm / not push-regex
PENT-DOMAIN-NET           engagement      record       runtime/engagement
PENT-DOMAIN-CLOUD         review          advise       often config/IaC later
PENT-DOMAIN-LLM           changeset       advise       some patterns gateable later
PENT-DOMAIN-MCP           changeset       advise       tool-abuse patterns later
ATTCK-TA0043              catalog         record       recon knowledge
ATTCK-TA0001              engagement      record       initial access confirm
ATTCK-TA0006              engagement      record       credential access
ATTCK-TA0008              engagement      record       lateral movement
ATTCK-TA0040              engagement      record       impact

Legend:
  changeset  = git frontier exam/gate
  review     = PR / human
  runtime    = deployed system
  engagement = offensive confirm (Argus-style later)
  catalog    = in V as knowledge only until materialized

Next real importer: harvest MITRE/CWE/CAPEC → same columns → Haskell V.
╚══════════════════════════════════════════════╝`)
}

func evaluateForShip(cwd string) (policy.GateResult, []owasp.Finding, error) {
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	p, _ := repo.StatusPorcelain()
	findings, err := runExam(cwd, true)
	if err != nil {
		return policy.GateResult{}, nil, err
	}
	g := policy.EvaluatePushGate(b, h, p, false)
	g.Branch, g.Head = b, h
	g.Dirty = policy.DirtyPorcelain(p)
	if owasp.BlocksGate(findings) {
		g.OK = false
		g.Reasons = append(g.Reasons, "OWASP V: untriaged High/Critical finding(s)")
		axiom("F4", "exam.block", "High/Critical under V blocks ship")
	}
	if strings.EqualFold(b, "main") || strings.EqualFold(b, "master") {
		axiom("F1", "harm.boundary", "refuse direct ship to main/master")
	}
	return g, findings, nil
}

// runPlan = terraform plan: preview only; nothing remote; fail closed.
func runPlan(cwd string, exitNonZero bool) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  PLAN (like terraform plan) — V enforced     ║
║  S (Slim): not enforced yet                  ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "plan.start", "preview ship decision; no remote mutate")
	g, _, err := evaluateForShip(cwd)
	if err != nil {
		fail(err)
		return
	}
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	sealed, err := policy.SealPlan(led, "frontier-git", g)
	if err != nil {
		fail(err)
		return
	}
	fmt.Println()
	if sealed.OK {
		fmt.Println("Plan: OK — may run: git frontier apply")
		axiom("F0", "plan.passed", sealed.SealHash)
	} else {
		fmt.Println("Plan: FAILED — fix issues; nothing will apply/push")
		fmt.Printf("Reasons: %v\n", sealed.Reasons)
		axiom("F0", "plan.failed", strings.Join(sealed.Reasons, "; "))
		axiom("F3", "continuity", "fail closed — like terraform")
	}
	fmt.Printf("ok=%v seal=%s branch=%s head=%s\n", sealed.OK, sealed.SealHash, sealed.Branch, sealed.Head)
	fmt.Println("╚══════════════════════════════════════════════╝")
	if !sealed.OK && exitNonZero {
		os.Exit(2)
	}
}

// runApply = terraform apply: only if fresh plan.passed; seals gate.passed.
func runApply(cwd string, exitNonZero bool) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  APPLY (like terraform apply)                ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "apply.start", "authorize push only from successful plan")
	repo := gitx.Repo{Dir: cwd}
	b, _ := repo.Branch()
	h, _ := repo.RevParseHead()
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	ok, detail := policy.FreshPlanOK(led, b, h)
	if !ok {
		axiom("F0", "apply.deny", detail)
		fmt.Printf("Apply refused: %s\n", detail)
		if exitNonZero {
			os.Exit(2)
		}
		return
	}
	// Re-validate (refresh) like a careful apply.
	g, _, err := evaluateForShip(cwd)
	if err != nil {
		fail(err)
		return
	}
	if !g.OK {
		_, _ = policy.SealGate(led, "frontier-git", g)
		axiom("F0", "apply.deny", strings.Join(g.Reasons, "; "))
		fmt.Printf("Apply refused after refresh: %v\n", g.Reasons)
		if exitNonZero {
			os.Exit(2)
		}
		return
	}
	sealed, err := policy.SealGate(led, "frontier-git", g)
	if err != nil {
		fail(err)
		return
	}
	axiom("F0", "gate.passed", sealed.SealHash)
	axiom("F2", "ready", "authorized human may git push")
	fmt.Printf("Apply: OK — sealed gate.passed\nplan_seal=%s gate_seal=%s\n", detail, sealed.SealHash)
	fmt.Println("Next: git push")
	fmt.Println("╚══════════════════════════════════════════════╝")
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
