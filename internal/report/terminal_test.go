package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/posit-dev/pev/internal/checks"
)

// TestRenderSkippedListsSkipsWithReason proves the --review-skipped view
// surfaces every SKIPPED check (and only those) with its skip reason,
// grouped under its category header, and never leaks PASS/FAIL rows.
func TestRenderSkippedListsSkipsWithReason(t *testing.T) {
	var buf bytes.Buffer
	RenderSkipped(&buf, fixtureReport(), false)
	out := buf.String()

	if !strings.Contains(out, "Skipped checks (1)") {
		t.Fatalf("missing skip header; got:\n%s", out)
	}
	if !strings.Contains(out, "[SKIP]") || !strings.Contains(out, "workbench.ssl.cert-key-match") {
		t.Fatalf("skipped check not listed; got:\n%s", out)
	}
	if !strings.Contains(out, "reason:") || !strings.Contains(out, "missing or invalid input") {
		t.Fatalf("skip reason not shown; got:\n%s", out)
	}
	// The failing and passing checks from the fixture must not appear.
	if strings.Contains(out, "net.egress.cdn-rstudio") || strings.Contains(out, "[FAIL]") {
		t.Fatalf("non-skip rows leaked into skip view; got:\n%s", out)
	}
	if strings.Contains(out, "[PASS]") {
		t.Fatalf("pass rows leaked into skip view; got:\n%s", out)
	}
}

// TestRenderSkippedEmpty proves the no-skips path prints a clear sentinel
// rather than an empty header.
func TestRenderSkippedEmpty(t *testing.T) {
	rep := fixtureReport()
	// Flip the lone skip to pass so nothing is skipped.
	for i := range rep.Results {
		if rep.Results[i].Status == checks.StatusSkip {
			rep.Results[i].Status = checks.StatusPass
		}
	}
	var buf bytes.Buffer
	RenderSkipped(&buf, rep, false)
	if got := buf.String(); !strings.Contains(got, "No checks were skipped.") {
		t.Fatalf("expected empty-skip sentinel; got:\n%s", got)
	}
}
