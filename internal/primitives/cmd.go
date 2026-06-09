package primitives

import (
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/system"
)

func init() {
	checks.Register("cmd", runCmd, []string{
		"cmd", "expect_exit", "expect_stdout_regex", "expect_stderr_regex",
		"timeout_seconds", "as_user", "skip_exit",
	})
}

// asUserWrap rewrites cmd so it runs as user u. Returns "" for the rewritten
// command when the run can't be performed safely (e.g. running as a
// different non-root user), in which case the caller should SKIP.
func asUserWrap(cmd, u string) (string, bool) {
	if u == "" {
		return cmd, true
	}
	current, err := user.Current()
	if err != nil {
		return cmd, true
	}
	if current.Username == u || current.Uid == u {
		return cmd, true
	}
	if os.Geteuid() != 0 {
		// Non-root, asked to impersonate someone else — can't.
		return "", false
	}
	// `runuser -u USER -- sh -c CMD` runs CMD with USER's environment and
	// PAM session. -- prevents argv leakage; sh -c lets the YAML keep
	// shell pipelines without us reinterpreting the body.
	quoted := strings.ReplaceAll(cmd, "'", `'\''`)
	return "runuser -u " + u + " -- sh -c '" + quoted + "'", true
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

	// `as_user` lets a check assert behavior for an unprivileged user
	// (e.g. `renv::install("renv")` succeeds without sudo). The
	// `unprivileged_user` input is set by the assess command; YAML
	// authors should write `as_user: "{{ .Inputs.unprivileged_user }}"`
	// so the user is consistent across checks.
	asUser, _ := getString(rc.Check.With, "as_user")
	wrapped, canRun := asUserWrap(cmd, asUser)
	if !canRun {
		return checks.Result{
			ID: rc.Check.ID, Title: rc.Check.Title,
			Status: checks.StatusSkip,
			Reason: "cannot impersonate " + asUser + " from a non-root user",
		}
	}
	cmd = wrapped

	res, err := system.RunCaptured(rc.Ctx, cmd, timeout)
	if err != nil {
		return unknownf(rc.Check, "command did not start: %v", err)
	}

	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title,
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

	// `skip_exit` lets a YAML author signal "this check is not applicable
	// on this host" from inside the script (e.g. "no apt lists present
	// yet"). Matching exit code → SKIP, with the script's stdout becoming
	// the human-readable reason. Checked before expect_exit so a script
	// can use a single exit code to mean either pass or skip.
	if skip, ok := getInt(rc.Check.With, "skip_exit"); ok && res.ExitCode == skip {
		r.Status = checks.StatusSkip
		reason := firstNonBlankLine(res.Stdout)
		if reason == "" {
			reason = firstNonBlankLine(res.Stderr)
		}
		if reason == "" {
			reason = fmt.Sprintf("script signalled skip (exit %d)", skip)
		}
		r.Reason = reason
		return r
	}

	if want, ok := getInt(rc.Check.With, "expect_exit"); ok && res.ExitCode != want {
		r.Status = checks.StatusFail
		r.Reason = fmt.Sprintf("exit %d != expected %d%s", res.ExitCode, want, outputSnippet(res.Stdout, res.Stderr))
		return r
	}
	if pat, ok := getString(rc.Check.With, "expect_stdout_regex"); ok && pat != "" {
		re, err := compileMultiline(pat)
		if err != nil {
			return unknownf(rc.Check, "invalid expect_stdout_regex: %v", err)
		}
		if !re.MatchString(res.Stdout) {
			r.Status = checks.StatusFail
			r.Reason = "stdout did not match expect_stdout_regex" + outputSnippet(res.Stdout, res.Stderr)
			return r
		}
	}
	if pat, ok := getString(rc.Check.With, "expect_stderr_regex"); ok && pat != "" {
		re, err := compileMultiline(pat)
		if err != nil {
			return unknownf(rc.Check, "invalid expect_stderr_regex: %v", err)
		}
		if !re.MatchString(res.Stderr) {
			r.Status = checks.StatusFail
			r.Reason = "stderr did not match expect_stderr_regex" + outputSnippet(res.Stdout, res.Stderr)
			return r
		}
	}

	r.Status = checks.StatusPass
	return r
}

// outputSnippet returns a short, single-line summary of the command's stdout
// and stderr, prefixed with ": ", suitable for tacking onto a fail Reason.
// Returns "" when both streams are empty. The full text is still kept on
// the Result's Evidence; this just makes the terminal/CI line useful at a
// glance ("exit 1 != expected 0: setfacl not installed").
func outputSnippet(stdout, stderr string) string {
	out := firstNonBlankLine(stdout)
	if out == "" {
		out = firstNonBlankLine(stderr)
	}
	if out == "" {
		return ""
	}
	if len(out) > 240 {
		out = out[:240] + "…"
	}
	return ": " + out
}

func firstNonBlankLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
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

// compileMultiline parses pat with the multi-line flag forced on so `^` /
// `$` anchors match line boundaries by default — the behavior YAML authors
// expect (`uname -m` outputs `x86_64\n`, and `^x86_64$` should match).
// Patterns may opt out by passing their own `(?-m)`.
func compileMultiline(pat string) (*regexp.Regexp, error) {
	if !strings.HasPrefix(pat, "(?") {
		pat = "(?m)" + pat
	}
	return regexp.Compile(pat)
}
