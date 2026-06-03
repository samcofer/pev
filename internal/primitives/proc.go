package primitives

import (
	"strings"
	"time"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/system"
)

func init() {
	checks.Register("proc", runProc, []string{"unit", "state"})
}

// runProc verifies a systemd unit is in the requested state (default active).
func runProc(rc checks.RunCtx) checks.Result {
	unit, ok := getString(rc.Check.With, "unit")
	if !ok || unit == "" {
		return unknownf(rc.Check, "missing required `unit` field")
	}
	state, _ := getString(rc.Check.With, "state")
	if state == "" {
		state = "active"
	}

	cmd := "systemctl is-" + state + " " + unit
	rc.CmdLog.Append(cmd)
	out, err := system.RunCaptured(rc.Ctx, cmd, 5*time.Second)
	if err != nil {
		return unknownf(rc.Check, "systemctl: %v", err)
	}
	got := strings.TrimSpace(out.Stdout)
	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title, Severity: rc.Check.Severity,
		Evidence: []checks.Evidence{{Command: cmd, Stdout: got, Stderr: out.Stderr}},
	}
	if got == state {
		r.Status = checks.StatusPass
		return r
	}
	r.Status = checks.StatusFail
	r.Reason = "unit " + unit + " is " + got + ", want " + state
	return r
}
