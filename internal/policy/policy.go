package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wadek/frontier-push-mcp/internal/ledger"
	"github.com/Wadek/frontier-push-mcp/internal/role"
)

// Deny > Ask > Allow evaluated in Require.
func Require(have role.Role, min role.Role, tool string) error {
	if !have.Can(min) {
		return fmt.Errorf("deny: tool %s needs role >= %s (have %s)", tool, min, have)
	}
	return nil
}

// GateResult is a sealed pre-push conscience check.
type GateResult struct {
	OK        bool           `json:"ok"`
	Reasons   []string       `json:"reasons"`
	Branch    string         `json:"branch"`
	Head      string         `json:"head"`
	Dirty     bool           `json:"dirty"`
	SealHash  string         `json:"seal_hash,omitempty"`
	ExpiresAt string         `json:"expires_at,omitempty"`
	Extras    map[string]any `json:"extras,omitempty"`
}

const gateTTL = 15 * time.Minute

// EvaluatePushGate is local-first and cheap: no model calls.
func EvaluatePushGate(branch, head, porcelain string, allowDirty bool) GateResult {
	var reasons []string
	dirty := strings.TrimSpace(porcelain) != ""
	if branch == "" {
		reasons = append(reasons, "detached HEAD or empty branch")
	}
	if head == "" {
		reasons = append(reasons, "missing HEAD")
	}
	if dirty && !allowDirty {
		reasons = append(reasons, "working tree dirty; commit or clean before push")
	}
	if strings.EqualFold(branch, "main") || strings.EqualFold(branch, "master") {
		reasons = append(reasons, "refusing direct push to main/master (use a feature branch)")
	}
	ok := len(reasons) == 0
	return GateResult{
		OK:      ok,
		Reasons: reasons,
		Branch:  branch,
		Head:    head,
		Dirty:   dirty,
	}
}

// SealGate writes gate.passed or gate.failed to the ledger.
func SealGate(l *ledger.Ledger, actor string, g GateResult) (*GateResult, error) {
	action := "gate.failed"
	if g.OK {
		action = "gate.passed"
		g.ExpiresAt = time.Now().UTC().Add(gateTTL).Format(time.RFC3339)
	}
	payload := map[string]any{
		"ok":         g.OK,
		"reasons":    g.Reasons,
		"branch":     g.Branch,
		"head":       g.Head,
		"dirty":      g.Dirty,
		"expires_at": g.ExpiresAt,
	}
	e, err := l.Append(actor, action, payload)
	if err != nil {
		return nil, err
	}
	g.SealHash = e.EntryHash
	return &g, nil
}

// FreshGateOK reports whether a recent gate.passed matches current head/branch.
func FreshGateOK(l *ledger.Ledger, branch, head string) (bool, string) {
	e, err := l.LastAction("gate.passed")
	if err != nil || e == nil {
		return false, "no gate.passed in ledger; run frontier_gate"
	}
	exp, _ := e.Payload["expires_at"].(string)
	if exp != "" {
		t, err := time.Parse(time.RFC3339, exp)
		if err == nil && time.Now().UTC().After(t) {
			return false, "gate expired; run frontier_gate again"
		}
	}
	b, _ := e.Payload["branch"].(string)
	h, _ := e.Payload["head"].(string)
	if b != branch || h != head {
		return false, "gate was for different branch/HEAD; re-run frontier_gate"
	}
	return true, e.EntryHash
}
