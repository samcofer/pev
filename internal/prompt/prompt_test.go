package prompt

import "testing"

func TestNonInteractiveTakesDefault(t *testing.T) {
	d := New(ModeNonInteractive)
	got, err := d.Input("workbench hostname:", "wb.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "wb.example.com" {
		t.Fatalf("want default, got %q", got)
	}
}

func TestYesModeTakesDefault(t *testing.T) {
	d := New(ModeYes)
	got, err := d.Confirm("Use SSL?", true)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("want true, got false")
	}
}

func TestInteractiveDowngradesWithoutTTY(t *testing.T) {
	// `go test` always runs without a TTY for stdin/stdout, so the
	// interactive driver must downgrade to yes-mode silently.
	d := New(ModeInteractive)
	got, err := d.Input("license file path:", "/var/lib/rstudio-server/x.lic")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/var/lib/rstudio-server/x.lic" {
		t.Fatalf("want default, got %q", got)
	}
}

// TestPasswordReturnsEmptyInNonInteractive locks in the secret-handling
// contract called out in CLAUDE.md §8 / cmd/assess.secretInputKeys: a
// password has no sensible auto-default, so callers must treat the
// empty return as "skip the dependent check" rather than fall through
// with a fake secret. ModeYes shares the same return so CI runs and
// piped sessions never end up holding a phony password.
func TestPasswordReturnsEmptyInNonInteractive(t *testing.T) {
	for _, mode := range []Mode{ModeYes, ModeNonInteractive} {
		d := New(mode)
		got, err := d.Password("PostgreSQL password:")
		if err != nil {
			t.Fatalf("mode=%v: %v", mode, err)
		}
		if got != "" {
			t.Fatalf("mode=%v: password should be empty, got %q (would leak as a real secret)", mode, got)
		}
	}
}

// TestInteractiveTTY-less downgrades to ModeYes (covered above) so
// Password() in interactive-mode-without-a-TTY MUST also return "".
// Belt-and-braces for the secret path: this is the path real CI
// pipelines exercise when they run pev without --non-interactive but
// without a controlling tty.
func TestPasswordReturnsEmptyInInteractiveWithoutTTY(t *testing.T) {
	d := New(ModeInteractive)
	got, err := d.Password("PostgreSQL password:")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("interactive-without-tty Password should return empty, got %q", got)
	}
}
