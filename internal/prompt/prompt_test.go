package prompt

import "testing"

// fakeDriver lets tests assert what would have been asked. We don't use it
// for the production driver's TTY behavior — that's exercised manually.
type fakeDriver struct{ answers map[string]string }

func (f *fakeDriver) Input(q, def string) (string, error) {
	if v, ok := f.answers[q]; ok {
		return v, nil
	}
	return def, nil
}
func (f *fakeDriver) Confirm(q string, def bool) (bool, error) { return def, nil }
func (f *fakeDriver) Select(q string, opts []string, def string) (string, error) {
	return def, nil
}

func (f *fakeDriver) MultiSelect(q string, opts, def []string) ([]string, error) {
	return def, nil
}

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
