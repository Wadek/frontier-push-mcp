package fronticli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Wadek/frontier-ship/internal/gitx"
	"github.com/Wadek/frontier-ship/internal/learn"
	"github.com/Wadek/frontier-ship/internal/ledger"
	"github.com/Wadek/frontier-ship/internal/owasp"
	"github.com/Wadek/frontier-ship/internal/policy"
	"github.com/Wadek/frontier-ship/internal/vscan"
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


// Version/Commit set via ldflags from cmd wrappers.
var (
	Version = "dev"
	Commit  = "none"
)

// Run executes frontier subcommands (V, S, plan, apply, ...).
// Git passthrough stays in cmd/frontier-git.
func Run(args []string) {
	if Version != "dev" || Commit != "none" {
		version, commit = Version, Commit
	}
	handleMeta(args)
}

// GuardPush is used by the git shim.
func GuardPush(soft bool) error { return guardPush(soft) }

// GuardCommit is used by the git shim.
func GuardCommit(args []string, soft, strict bool) error {
	return guardCommit(args, soft, strict)
}

// FindRealGit exposes real git path for the shim.
func FindRealGit() string { return findRealGit() }

// RunGitPassthrough runs real git and returns exit code.
func RunGitPassthrough(git string, args []string) int {
	return runPassthrough(git, args)
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

  git frontier L            Learn / Landscape (classify this project)
  git frontier L classify [path]
  git frontier V            run Vulnerabilities exam (OWASP built-in)
  git frontier V list       list programmatic scanners (owasp, checkov, …)
  git frontier V checkov    run Checkov adapter if installed (no tokens)
  git frontier S            Slim / vibe-bloat (PLANNED — not enforced)
  git frontier exam         alias of V
  git frontier enhance V    programmatic pack + lean brief for host model
  git frontier enhance status|seal
  git frontier mock-import  mock V-importer list

  git frontier status|ledger|demo|explain

Env: FRONTIER_SOFT=1  FRONTIER_VERBOSE=1  FRONTIER_GIT_BIN  FRONTIER_LEDGER
     FRONTIER_V_AUTO=1  also run available adapters during enhance/V pack

Nothing remote goes if plan/apply fails (like terraform).

Same commands as standalone:  frontier L | V | enhance V | plan | apply | S
(Not \"go frontier\" — go is the Go toolchain; use frontier or go run ./cmd/frontier)`)
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
	case "L", "l", "learn":
		runLearn(cwd, args[1:])
	case "V", "v", "exam":
		if len(args) > 1 {
			runVSub(cwd, args[1:])
			return
		}
		runExam(cwd, true)
	case "enhance":
		runEnhance(cwd, args[1:])
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
  L  Learn / Landscape — ingest + classify before change (first phase of S)
  V  Vulnerabilities — security definitions (OWASP…); enforced at changeset
  S  Slim — vibe-code bloat reduction; PLANNED (not enforced yet)

Enhance:
  git frontier enhance V   # programmatic first, lean brief for host model (Grok/Fable/…)

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
  First:   git frontier L classify   (learn before slim)

  Stick with:  git frontier L | V | plan | apply | push
╚══════════════════════════════════════════════╝`)
}

func runLearn(cwd string, args []string) {
	if len(args) == 0 {
		runLearnClassify(cwd)
		return
	}
	switch strings.ToLower(args[0]) {
	case "classify":
		root := cwd
		if len(args) > 1 {
			root = args[1]
		}
		runLearnClassify(root)
	case "status":
		runLearnStatus(cwd)
	default:
		fmt.Fprintf(os.Stderr, "unknown L subcommand %q (try: frontier L classify [path])\n", args[0])
		os.Exit(2)
	}
}

func runLearnClassify(root string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  L CLASSIFY — learn before change (no mutate)║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "learn.start", "programmatic landscape; no remote mutate")
	ls, err := learn.Classify(root)
	if err != nil {
		fail(err)
		return
	}
	art, err := learn.WriteArtifacts(ls)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("name:       %s\n", ls.Name)
	fmt.Printf("kind:       %s (%s)\n", ls.Kind, ls.Confidence)
	fmt.Printf("git:        %v\n", ls.HasGit)
	fmt.Printf("compose:    %v %v\n", ls.HasCompose, ls.ComposeServices)
	fmt.Printf("files~:     %d\n", ls.FileCountApprox)
	if len(ls.Reasons) > 0 {
		fmt.Printf("why:        %s\n", strings.Join(ls.Reasons, "; "))
	}
	fmt.Printf("\nbrief:  %s\njson:   %s\n", art.Markdown, art.JSON)

	led, err := ledger.Open(findLedger(root))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "learn.classified", map[string]any{
		"root":       ls.Root,
		"name":       ls.Name,
		"kind":       ls.Kind,
		"confidence": ls.Confidence,
		"has_git":    ls.HasGit,
		"has_compose": ls.HasCompose,
		"brief":      art.Markdown,
		"json":       art.JSON,
	})
	axiom("F0", "ledger.append", "learn.classified sealed")
	axiom("F4", "learn.done", ls.Kind)
}

func runLearnStatus(cwd string) {
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	rows, _ := led.Tail(40)
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Action, "learn.") {
			fmt.Printf("%d %s %s %v\n", r.Seq, r.TS, r.Action, r.Payload)
			n++
		}
	}
	latest := filepath.Join(cwd, ".frontier", "learn", "LATEST")
	if b, err := os.ReadFile(latest); err == nil {
		fmt.Printf("LATEST artifacts: %s\n", strings.TrimSpace(string(b)))
	}
	if n == 0 {
		fmt.Println("no learn.* seals yet — run: frontier L classify")
	}
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

func runVSub(cwd string, args []string) {
	if len(args) == 0 {
		runExam(cwd, true)
		return
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		fmt.Println(`╔══════════════════════════════════════════════╗
║     FRONTIER V — programmatic scanners       ║
╚══════════════════════════════════════════════╝
name        builtin  available  notes
----        -------  ---------  -----`)
		for _, s := range vscan.Registry() {
			avail := "no"
			if s.Available() {
				avail = "yes"
			}
			builtin := "no"
			if s.Builtin() {
				builtin = "yes"
			}
			note := ""
			if !s.Available() && !s.Builtin() {
				note = "install or planned"
			}
			if s.Name() == "checkov" && !s.Available() {
				note = "pip install checkov"
			}
			fmt.Printf("%-11s %-7s  %-9s  %s\n", s.Name(), builtin, avail, note)
		}
		fmt.Println(`
Gate/plan still hard-block only on built-in owasp-v0 High/Critical.
Adapters enrich V / enhance briefs without burning model tokens.`)
	default:
		sc, ok := vscan.Lookup(sub)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown V scanner %q (try: frontier V list)\n", sub)
			os.Exit(2)
		}
		fmt.Printf("╔══════════════════════════════════════════════╗\n║     FRONTIER V — adapter %-12s       ║\n╚══════════════════════════════════════════════╝\n", sc.Name())
		axiom("F4", "exam.adapter", sc.Name())
		res, err := sc.Scan(cwd)
		if err != nil {
			fail(err)
			return
		}
		if res.Skipped {
			fmt.Printf("skipped: %s\n", res.SkipWhy)
			return
		}
		fmt.Printf("source: %s\nfindings: %d\n", res.Source, len(res.Findings))
		for _, f := range res.Findings {
			fmt.Printf("  [%s] %s %s:%d  %s\n", f.Severity, f.RuleID, f.Path, f.Line, f.Snippet)
		}
		led, err := ledger.Open(findLedger(cwd))
		if err == nil {
			_, _ = led.Append("frontier-git", "exam.adapter", map[string]any{
				"source": res.Source, "findings": len(res.Findings), "meta": res.Meta,
			})
			axiom("F0", "ledger.append", "exam.adapter sealed")
		}
	}
}

func runEnhance(cwd string, args []string) {
	if len(args) == 0 {
		fmt.Println(`enhance commands:
  frontier enhance V         programmatic V pack + lean host brief
  frontier enhance status    last enhance.* ledger seals
  frontier enhance seal PATH ingest host-model result JSON (advise)`)
		return
	}
	switch strings.ToLower(args[0]) {
	case "V", "v":
		runEnhanceV(cwd)
	case "status":
		runEnhanceStatus(cwd)
	case "seal":
		if len(args) < 2 {
			fail(fmt.Errorf("usage: frontier enhance seal <result.json>"))
			return
		}
		runEnhanceSeal(cwd, args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown enhance subcommand %q\n", args[0])
		os.Exit(2)
	}
}

func runEnhanceV(cwd string) {
	fmt.Println(`╔══════════════════════════════════════════════╗
║  ENHANCE V — programmatic first, then host   ║
╚══════════════════════════════════════════════╝`)
	axiom("F0", "enhance.start", "build pack without tokens; hand residual to host model")
	opts := vscan.Options{}
	// Explicit adapters on enhance: checkov when available (still programmatic).
	if s, ok := vscan.Lookup("checkov"); ok && s.Available() {
		opts.Adapters = []string{"checkov"}
	}
	pack, err := vscan.BuildPack(cwd, opts)
	if err != nil {
		fail(err)
		return
	}
	art, err := vscan.WriteEnhanceBrief(cwd, pack)
	if err != nil {
		fail(err)
		return
	}
	fmt.Printf("disposition (programmatic): %s\n", pack.Disposition)
	fmt.Printf("findings: %d (brief shows <=%d)\n", len(pack.Findings), vscan.MaxBriefFindings)
	fmt.Printf("adapters: %s\n", strings.Join(pack.AdaptersRun, ", "))
	fmt.Printf("scope: %s\n", pack.ScopeMode)
	fmt.Printf("\nbrief:  %s\njson:   %s\n", art.Markdown, art.JSON)
	fmt.Println("\nHost (Grok / Fable / …): read the brief. Do residual work only. Then:")
	fmt.Println("  frontier enhance seal .frontier/enhance/<result>.json")
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "enhance.requested", map[string]any{
		"control":               "changeset",
		"disposition":           pack.Disposition,
		"findings_programmatic": len(pack.Findings),
		"brief":                 art.Markdown,
		"json":                  art.JSON,
		"adapters":              pack.AdaptersRun,
		"token_budget": map[string]int{
			"max_findings": vscan.MaxBriefFindings,
			"max_paths":    vscan.MaxBriefPaths,
			"max_bytes":    vscan.MaxBriefBytes,
		},
	})
	axiom("F0", "ledger.append", "enhance.requested sealed")
	axiom("F4", "enhance.handoff", "waiting on host model — no gate change")
}

func runEnhanceStatus(cwd string) {
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	rows, _ := led.Tail(40)
	n := 0
	for _, r := range rows {
		if strings.HasPrefix(r.Action, "enhance.") {
			fmt.Printf("%d %s %s %v\n", r.Seq, r.TS, r.Action, r.Payload)
			n++
		}
	}
	if n == 0 {
		fmt.Println("no enhance.* seals yet — run: frontier enhance V")
	}
}

func runEnhanceSeal(cwd, path string) {
	payload, err := vscan.ReadSealFile(path)
	if err != nil {
		fail(err)
		return
	}
	disp := payload.DispositionSuggest
	if disp == "" || disp == "block" {
		// Enhance never auto-blocks gate; promote into V definitions instead.
		if disp == "block" {
			axiom("F4", "enhance.advise_only", "host suggested block — sealed as advise until V promotion")
		}
		disp = "advise"
	}
	led, err := ledger.Open(findLedger(cwd))
	if err != nil {
		fail(err)
		return
	}
	_, _ = led.Append("frontier-git", "enhance.completed", map[string]any{
		"summary":             payload.Summary,
		"findings":            len(payload.Findings),
		"disposition_suggest": disp,
		"tools_used":          payload.ToolsUsed,
		"residual_risk":       payload.ResidualRisk,
		"seal_path":           path,
		"blocks_gate":         false,
	})
	axiom("F0", "ledger.append", "enhance.completed sealed (advise)")
	fmt.Printf("enhance sealed: disposition=%s findings=%d\n", disp, len(payload.Findings))
	fmt.Println("note: does not change gate — promote durable rules into V to block.")
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
