package primitives

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/discover"
	"github.com/posit-dev/pev/internal/logging"
)

func runRC(t *testing.T, c checks.Check, facts discover.HostFacts) checks.Result {
	t.Helper()
	cl, _ := logging.NewCmdLog("", "host", false) // no-op
	rc := checks.RunCtx{Ctx: context.Background(), Check: c, Facts: facts, CmdLog: cl}
	r, err := checks.Lookup(c.Primitive)
	if err != nil {
		t.Fatal(err)
	}
	return r(rc)
}

func TestCmdPrimitivePassFail(t *testing.T) {
	pass := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityInfo, Primitive: "cmd",
		With: map[string]interface{}{"cmd": "true", "expect_exit": 0},
	}
	if r := runRC(t, pass, discover.HostFacts{}); r.Status != checks.StatusPass {
		t.Fatalf("expected pass, got %s/%s", r.Status, r.Reason)
	}
	fail := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityInfo, Primitive: "cmd",
		With: map[string]interface{}{"cmd": "false", "expect_exit": 0},
	}
	if r := runRC(t, fail, discover.HostFacts{}); r.Status != checks.StatusFail {
		t.Fatalf("expected fail, got %s", r.Status)
	}
}

func TestFilePrimitiveExistence(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "exists")
	if err := os.WriteFile(tmp, []byte("ssl-enabled=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityInfo, Primitive: "file",
		With: map[string]interface{}{"path": tmp, "must_exist": true, "contains_regex": "(?m)^ssl-enabled=1"},
	}
	if r := runRC(t, c, discover.HostFacts{}); r.Status != checks.StatusPass {
		t.Fatalf("got %s/%s", r.Status, r.Reason)
	}

	missing := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityInfo, Primitive: "file",
		With: map[string]interface{}{"path": "/nonexistent/path"},
	}
	if r := runRC(t, missing, discover.HostFacts{}); r.Status != checks.StatusFail {
		t.Fatalf("got %s/%s", r.Status, r.Reason)
	}
}

func TestSizingPrimitive(t *testing.T) {
	pass := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityWarning, Primitive: "sizing",
		With: map[string]interface{}{"cpus_min": 1, "mem_gb_min": 1, "disk_gb_min": map[string]interface{}{"/": 0}},
	}
	r := runRC(t, pass, discover.HostFacts{CPUs: 4, MemMB: 8192, DiskGB: map[string]int{"/": 100}})
	if r.Status != checks.StatusPass {
		t.Fatalf("expected pass, got %s/%s", r.Status, r.Reason)
	}
	fail := checks.Check{
		ID: "x", Title: "x", Severity: checks.SeverityWarning, Primitive: "sizing",
		With: map[string]interface{}{"cpus_min": 100},
	}
	r = runRC(t, fail, discover.HostFacts{CPUs: 4, MemMB: 8192, DiskGB: map[string]int{"/": 100}})
	if r.Status != checks.StatusFail || !strings.Contains(r.Reason, "cpus") {
		t.Fatalf("expected cpu failure, got %s/%s", r.Status, r.Reason)
	}
}
