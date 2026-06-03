package checks

import "testing"

func TestFilter(t *testing.T) {
	in := []Check{
		{ID: "a", Severity: SeverityInfo, Tags: []string{"net"}, AppliesTo: AppliesTo{Products: []string{"workbench"}}},
		{ID: "b", Severity: SeverityWarning, Tags: []string{"net", "egress"}, AppliesTo: AppliesTo{Products: []string{"connect"}}},
		{ID: "c", Severity: SeverityBlocking, Tags: []string{"license"}},
		{ID: "d", Severity: SeverityWarning, Tags: []string{"experimental"}},
	}

	t.Run("severity-min", func(t *testing.T) {
		got := Filter{SeverityMin: SeverityWarning}.Apply(in)
		if got, want := len(got), 3; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})
	t.Run("must-have-tags", func(t *testing.T) {
		got := Filter{Tags: []string{"net", "egress"}}.Apply(in)
		if len(got) != 1 || got[0].ID != "b" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("skip-tags", func(t *testing.T) {
		got := Filter{SkipTags: []string{"experimental"}}.Apply(in)
		if len(got) != 3 {
			t.Fatalf("got %d", len(got))
		}
	})
	t.Run("skip-ids", func(t *testing.T) {
		got := Filter{SkipIDs: []string{"a", "c"}}.Apply(in)
		if len(got) != 2 {
			t.Fatalf("got %d", len(got))
		}
	})
	t.Run("products-overlap", func(t *testing.T) {
		got := Filter{Products: []string{"workbench"}}.Apply(in)
		// 'a' has products=workbench; 'c' and 'd' have no products gate (passthrough); 'b' is connect-only.
		ids := map[string]bool{}
		for _, c := range got {
			ids[c.ID] = true
		}
		if !ids["a"] || ids["b"] || !ids["c"] || !ids["d"] {
			t.Fatalf("filter wrong: %+v", ids)
		}
	})
}
