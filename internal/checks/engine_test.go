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

// TestMissingRequires covers the seven tokens the requires gate
// understands plus the "unknown token defaults to satisfied" contract.
func TestMissingRequires(t *testing.T) {
	cases := []struct {
		name     string
		requires []string
		hf       discover.HostFacts
		want     string // empty = all requirements satisfied
	}{
		{"empty list", nil, discover.HostFacts{}, ""},

		{"r missing", []string{"r"}, discover.HostFacts{}, "r"},
		{"r present", []string{"r"}, discover.HostFacts{R: []string{"4.4.1"}}, ""},

		{"python missing", []string{"python"}, discover.HostFacts{}, "python"},
		{"python present", []string{"python"}, discover.HostFacts{Python: []string{"3.12"}}, ""},

		{"quarto missing", []string{"quarto"}, discover.HostFacts{}, "quarto"},
		{"quarto present", []string{"quarto"}, discover.HostFacts{Quarto: []string{"1.5.0"}}, ""},

		{"uv missing", []string{"uv"}, discover.HostFacts{}, "uv"},
		{"uv present", []string{"uv"}, discover.HostFacts{HasUV: true}, ""},

		{"pip missing", []string{"pip"}, discover.HostFacts{}, "pip"},
		{"pip present", []string{"pip"}, discover.HostFacts{HasPip: true}, ""},

		{"apt missing", []string{"apt"}, discover.HostFacts{}, "apt"},
		{"apt present", []string{"apt"}, discover.HostFacts{HasApt: true}, ""},

		{"dnf missing", []string{"dnf"}, discover.HostFacts{}, "dnf"},
		{"dnf present", []string{"dnf"}, discover.HostFacts{HasDNF: true}, ""},

		// First-missing-wins: r missing AND python missing → "r"
		// surfaces because it appears first in the requires slice.
		// Stable order makes the SKIP reason deterministic.
		{"first missing wins",
			[]string{"r", "python"},
			discover.HostFacts{},
			"r",
		},
		{"unknown token defaults to satisfied",
			[]string{"future-token-that-does-not-exist"},
			discover.HostFacts{},
			"",
		},
		{"all satisfied",
			[]string{"r", "python", "uv", "apt"},
			discover.HostFacts{
				R: []string{"4.4"}, Python: []string{"3.12"},
				HasUV: true, HasApt: true,
			},
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := missingRequires(tc.requires, tc.hf); got != tc.want {
				t.Fatalf("missingRequires(%v) = %q, want %q", tc.requires, got, tc.want)
			}
		})
	}
}
