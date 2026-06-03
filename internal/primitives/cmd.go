package primitives

import (
	"regexp"
	"time"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/system"
)

func init() {
	checks.Register("cmd", runCmd, []string{
		"cmd", "expect_exit", "expect_stdout_regex", "expect_stderr_regex",
		"timeout_seconds",
	})
}

// runCmd executes a shell command and matches its output. Useful when an SE
// would naturally type the same command at the terminal.
func runCmd(rc checks.RunCtx) checks.Result {
	cmd, ok := getString(rc.Check.With, "cmd")
	if !ok || cmd == "" {
		return unknownf(rc.Check, "missing required `cmd` field")
	}
	timeout := 30 * time.Second
	if t, ok := getInt(rc.Check.With, "timeout_seconds"); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	res, err := system.RunCaptured(rc.Ctx, cmd, timeout)
	if err != nil {
		return unknownf(rc.Check, "command did not start: %v", err)
	}
	rc.CmdLog.Append(cmd)

	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title, Severity: rc.Check.Severity,
		Evidence: []checks.Evidence{{
			Command: cmd, Stdout: truncate(res.Stdout, 4096),
			Stderr: truncate(res.Stderr, 1024),
			Note:   noteFor(res),
		}},
	}

	if res.TimedOut {
		r.Status = checks.StatusFail
		r.Reason = "command timed out"
		return r
	}

	if want, ok := getInt(rc.Check.With, "expect_exit"); ok && res.ExitCode != want {
		r.Status = checks.StatusFail
		r.Reason = "exit code != expected"
		return r
	}
	if pat, ok := getString(rc.Check.With, "expect_stdout_regex"); ok && pat != "" {
		re, err := regexp.Compile(pat)
		if err != nil {
			return unknownf(rc.Check, "invalid expect_stdout_regex: %v", err)
		}
		if !re.MatchString(res.Stdout) {
			r.Status = checks.StatusFail
			r.Reason = "stdout did not match expect_stdout_regex"
			return r
		}
	}
	if pat, ok := getString(rc.Check.With, "expect_stderr_regex"); ok && pat != "" {
		re, err := regexp.Compile(pat)
		if err != nil {
			return unknownf(rc.Check, "invalid expect_stderr_regex: %v", err)
		}
		if !re.MatchString(res.Stderr) {
			r.Status = checks.StatusFail
			r.Reason = "stderr did not match expect_stderr_regex"
			return r
		}
	}

	r.Status = checks.StatusPass
	return r
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…(truncated)"
}

func noteFor(r system.CommandResult) string {
	if r.TimedOut {
		return "timed out"
	}
	return ""
}
