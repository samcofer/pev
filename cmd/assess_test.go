package cmd

import (
	"reflect"
	"testing"
)

// TestRedactSecretsScrubsKnownSecretKeys locks in the on-disk-redaction
// contract (CLAUDE.md §8): every secret-shaped input key must be
// replaced with "(redacted)" before the inputs map lands in the JSON
// report — but the original map kept by the engine retains the raw
// secret so primitives can still template against it.
func TestRedactSecretsScrubsKnownSecretKeys(t *testing.T) {
	in := map[string]string{
		"postgres_host":     "db.example.com",
		"postgres_port":     "5432",
		"postgres_user":     "rstudio",
		"postgres_password": "supersecret",
		"connect_smtp_host": "smtp.example.com",
	}
	out := redactSecrets(in)
	if out["postgres_password"] != "(redacted)" {
		t.Fatalf("password was not redacted: %q", out["postgres_password"])
	}
	if out["postgres_host"] != "db.example.com" {
		t.Fatalf("non-secret host was rewritten: %q", out["postgres_host"])
	}
	// Original map still holds the raw secret.
	if in["postgres_password"] != "supersecret" {
		t.Fatalf("original input map was mutated; the engine needs the raw secret")
	}
}

// TestRedactSecretsLeavesEmptySecretAsIs guards against accidentally
// emitting "(redacted)" for an unset secret — that would mislead the
// reader into thinking a password WAS supplied. Empty stays empty.
func TestRedactSecretsLeavesEmptySecretAsIs(t *testing.T) {
	in := map[string]string{"postgres_password": ""}
	out := redactSecrets(in)
	if out["postgres_password"] != "" {
		t.Fatalf("empty secret should stay empty, got %q", out["postgres_password"])
	}
}

// TestFilterNoneOption locks in the multi-select tiebreak: real products
// always beat the "no products" sentinel because the menu pre-selects
// the sentinel and most SEs add a product without un-ticking it first.
// Sentinel-alone produces an empty slice; the caller maps that to the
// "none" string via len(selected) == 0.
func TestFilterNoneOption(t *testing.T) {
	const sentinel = "system configuration checks - product independent"
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"sentinel alone leaves no real picks", []string{sentinel}, []string{}},
		{
			"real products beat sentinel",
			[]string{sentinel, "workbench"},
			[]string{"workbench"},
		},
		{
			"sentinel mixed with multiple products",
			[]string{"workbench", sentinel, "connect"},
			[]string{"workbench", "connect"},
		},
		{
			"no sentinel, all real",
			[]string{"workbench", "connect"},
			[]string{"workbench", "connect"},
		},
		{"empty input", []string{}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterNoneOption(tc.in, sentinel)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), tc.want, len(tc.want))
			}
			if len(tc.want) > 0 && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestWantOutputs covers the small flag-parsing helper that decides
// which report formats to emit. No --output flag → both; explicit list
// → just those.
func TestWantOutputs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		md   bool
		j    bool
	}{
		{"empty defaults to both", nil, true, true},
		{"md only", []string{"md"}, true, false},
		{"json only", []string{"json"}, false, true},
		{"markdown alias", []string{"markdown"}, true, false},
		{"both explicit", []string{"md", "json"}, true, true},
		{"trims whitespace", []string{" md ", "  json"}, true, true},
		{"unknown silently dropped", []string{"pdf"}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md, j := wantOutputs(tc.in)
			if md != tc.md || j != tc.j {
				t.Fatalf("got md=%v json=%v, want md=%v json=%v", md, j, tc.md, tc.j)
			}
		})
	}
}
