package checks

import (
	"context"
	"testing"

	"github.com/posit-dev/pev/internal/discover"
)

// fakePrimitive registers a primitive that echoes back its `with.expect` value
// as the Result.Status — handy for engine unit tests.
func init() {
	Register("test-fake", func(rc RunCtx) Result {
		want, _ := rc.Check.With["expect"].(string)
		return Result{Status: Status(want)}
	}, []string{"expect", "name"})
}

func TestEngineSkipsOnAppliesTo(t *testing.T) {
	e := Engine{Facts: discover.HostFacts{OS: "rhel-9"}}
	c := Check{
		ID: "x", Title: "x", Why: "x", Primitive: "test-fake",
		AppliesTo: AppliesTo{OS: []string{"ubuntu-22.04"}}, With: map[string]interface{}{"expect": "pass"},
	}
	r := e.runOne(context.Background(), c)
	if r.Status != StatusSkip {
		t.Fatalf("want skip, got %s", r.Status)
	}
}

func TestEngineExpandsTemplate(t *testing.T) {
	e := Engine{
		Facts:  discover.HostFacts{Hostname: "h1"},
		Inputs: map[string]string{"workbench_hostname": "wb.example"},
	}
	c := Check{
		ID: "x", Title: "x", Why: "x", Primitive: "test-fake",
		With: map[string]interface{}{"name": "{{ .Inputs.workbench_hostname }}", "expect": "pass"},
	}
	r := e.runOne(context.Background(), c)
	if r.Status != StatusPass {
		t.Fatalf("want pass, got %s reason=%s", r.Status, r.Reason)
	}
}

func TestEngineSkipsOnMissingInput(t *testing.T) {
	e := Engine{Facts: discover.HostFacts{}}
	c := Check{
		ID: "x", Title: "x", Why: "x", Primitive: "test-fake",
		With: map[string]interface{}{"name": "{{ .Inputs.missing }}", "expect": "pass"},
	}
	r := e.runOne(context.Background(), c)
	if r.Status != StatusSkip {
		t.Fatalf("want skip, got %s", r.Status)
	}
}
