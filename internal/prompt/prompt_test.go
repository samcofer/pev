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
