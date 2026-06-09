package report

import (
	"testing"

	"github.com/posit-dev/pev/internal/checks"
)

func TestComputeRegressionAndImprovement(t *testing.T) {
	a := checks.Report{SchemaVersion: 1, Results: []checks.Result{
		{ID: "a", Status: checks.StatusPass},
		{ID: "b", Status: checks.StatusFail},
		{ID: "c", Status: checks.StatusPass},
	}}
	b := checks.Report{SchemaVersion: 1, Results: []checks.Result{
		{ID: "a", Status: checks.StatusFail}, // regression
		{ID: "b", Status: checks.StatusPass}, // improvement
		{ID: "c", Status: checks.StatusPass}, // unchanged
		{ID: "d", Status: checks.StatusPass}, // added
	}}
	d, err := Compute(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if !d.HasRegressions() {
		t.Fatal("expected regression flag set")
	}
	if len(d.Regressions) != 1 || d.Regressions[0].ID != "a" {
		t.Fatalf("regressions: %+v", d.Regressions)
	}
	if len(d.Improvements) != 1 || d.Improvements[0].ID != "b" {
		t.Fatalf("improvements: %+v", d.Improvements)
	}
	if len(d.Added) != 1 || d.Added[0].ID != "d" {
		t.Fatalf("added: %+v", d.Added)
	}
}

func TestComputeRejectsSchemaMismatch(t *testing.T) {
	a := checks.Report{SchemaVersion: 1}
	b := checks.Report{SchemaVersion: 2}
	if _, err := Compute(a, b); err == nil {
		t.Fatal("expected schema mismatch error")
	}
}
