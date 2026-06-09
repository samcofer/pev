package primitives

import (
	"runtime"
	"testing"

	"github.com/posit-dev/pev/internal/checks"
	"github.com/posit-dev/pev/internal/discover"
)

func TestSysctlPrimitive(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sysctl primitive requires /proc/sys; linux-only")
	}
	// kernel.osrelease always exists on Linux and is a non-empty string;
	// we don't assert a numeric range, just that the read succeeds.
	c := checks.Check{
		ID: "x", Title: "x", Primitive: "sysctl",
		With: map[string]interface{}{"key": "kernel.osrelease"},
	}
	r := runRC(t, c, discover.HostFacts{})
	if r.Status != checks.StatusPass {
		t.Fatalf("expected pass on kernel.osrelease, got %s/%s", r.Status, r.Reason)
	}

	missing := checks.Check{
		ID: "x", Title: "x", Primitive: "sysctl",
		With: map[string]interface{}{"key": "kernel.this_does_not_exist"},
	}
	r = runRC(t, missing, discover.HostFacts{})
	if r.Status != checks.StatusUnknown {
		t.Fatalf("expected unknown on missing key, got %s", r.Status)
	}
}
