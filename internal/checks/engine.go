package checks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"text/template"
	"time"

	"github.com/posit-dev/pev/internal/discover"
	"github.com/posit-dev/pev/internal/system"
)

// Engine runs a list of Checks against a HostFacts snapshot and a set of inputs.
type Engine struct {
	Facts  discover.HostFacts
	Inputs map[string]string

	// Progress, when non-nil, receives one line per check as it runs.
	// Format: "[i/N] running <id> ...". Useful for stderr output during
	// long-running primitives (apt update, uv venv, renv install) so the
	// SE knows pev hasn't hung. Final per-check status is NOT emitted —
	// that lives in the report.
	Progress io.Writer
}

// Run filters checks by AppliesTo, gates root-only checks, expands templates
// in `with:`, dispatches to the registered primitive, and returns Results
// sorted by ID.
func (e *Engine) Run(ctx context.Context, all []Check) []Result {
	out := make([]Result, 0, len(all))
	for i, c := range all {
		if e.Progress != nil {
			fmt.Fprintf(e.Progress, "[%d/%d] %s\n", i+1, len(all), c.ID)
		}
		out = append(out, e.runOne(ctx, c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (e *Engine) runOne(ctx context.Context, c Check) Result {
	start := time.Now()
	base := Result{
		ID: c.ID, Title: c.Title,
		Why: c.Why, References: c.References,
	}
	finish := func(r Result) Result {
		r.DurationMS = time.Since(start).Milliseconds()
		return r
	}

	if !appliesTo(c.AppliesTo, e.Facts) {
		base.Status = StatusSkip
		base.Reason = "does not apply to this host"
		return finish(base)
	}
	if missing := missingRequires(c.AppliesTo.Requires, e.Facts); missing != "" {
		base.Status = StatusSkip
		base.Reason = "missing required tooling: " + missing
		return finish(base)
	}
	if c.RequiresRoot && !system.IsRoot() {
		base.Status = StatusSkip
		base.Reason = "requires root; rerun with sudo to evaluate"
		return finish(base)
	}

	expanded, err := expandWith(c.With, e.Facts, e.Inputs)
	if err != nil {
		base.Status = StatusSkip
		base.Reason = "missing or invalid input: " + err.Error()
		return finish(base)
	}
	c.With = expanded

	runner, err := Lookup(c.Primitive)
	if err != nil {
		base.Status = StatusUnknown
		base.Reason = err.Error()
		return finish(base)
	}

	r := runner(RunCtx{
		Ctx: ctx, Check: c, Facts: e.Facts, Inputs: e.Inputs,
	})
	r.ID = c.ID
	r.Title = c.Title
	r.Category = CategoryFor(c)
	r.Why = c.Why
	r.References = c.References
	if r.Status == StatusFail || r.Status == StatusUnknown {
		r.Remediation = c.Remediation
	}
	return finish(r)
}

// appliesTo matches a Check's AppliesTo gate against host facts. An empty
// list for a dimension is "any" — only non-empty lists are restrictive.
//
// Product gating is intentionally absent here: it lives in Filter.Apply
// (internal/checks/filter.go), which the assess command runs before
// handing the surviving checks to the engine.
func appliesTo(a AppliesTo, hf discover.HostFacts) bool {
	if len(a.OS) > 0 && !contains(a.OS, hf.OS) {
		return false
	}
	if len(a.Arch) > 0 && !contains(a.Arch, hf.Arch) {
		return false
	}
	return true
}

func contains(set []string, s string) bool {
	for _, e := range set {
		if e == s {
			return true
		}
	}
	return false
}

// missingRequires returns the first required-fact token that the host
// doesn't satisfy, or "" when every requirement is met. Unknown tokens
// are treated as a satisfied fact so adding a new token to a YAML pack
// doesn't blanket-SKIP the catalog on older pev binaries.
func missingRequires(requires []string, hf discover.HostFacts) string {
	for _, req := range requires {
		switch req {
		case "r":
			if len(hf.R) == 0 {
				return req
			}
		case "python":
			if len(hf.Python) == 0 {
				return req
			}
		case "quarto":
			if len(hf.Quarto) == 0 {
				return req
			}
		case "uv":
			if !hf.HasUV {
				return req
			}
		case "pip":
			if !hf.HasPip {
				return req
			}
		case "apt":
			if !hf.HasApt {
				return req
			}
		case "dnf":
			if !hf.HasDNF {
				return req
			}
		}
	}
	return ""
}

// expandWith renders any string-typed value in the `with:` payload through
// text/template against {{ .Facts }} and {{ .Inputs.X }}. Missing keys cause
// an error so the engine can surface them as a SKIP with a clear reason.
func expandWith(with map[string]interface{}, facts discover.HostFacts, inputs map[string]string) (map[string]interface{}, error) {
	if len(with) == 0 {
		return with, nil
	}
	out := make(map[string]interface{}, len(with))
	data := struct {
		Facts  discover.HostFacts
		Inputs map[string]string
	}{Facts: facts, Inputs: inputs}

	var walk func(v interface{}) (interface{}, error)
	walk = func(v interface{}) (interface{}, error) {
		switch x := v.(type) {
		case string:
			if !looksLikeTemplate(x) {
				return x, nil
			}
			t, err := template.New("with").Option("missingkey=error").Parse(x)
			if err != nil {
				return nil, fmt.Errorf("template parse: %w", err)
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				return nil, fmt.Errorf("template execute: %w", err)
			}
			return buf.String(), nil
		case map[string]interface{}:
			m := make(map[string]interface{}, len(x))
			for k, vv := range x {
				nv, err := walk(vv)
				if err != nil {
					return nil, err
				}
				m[k] = nv
			}
			return m, nil
		case []interface{}:
			s := make([]interface{}, len(x))
			for i, vv := range x {
				nv, err := walk(vv)
				if err != nil {
					return nil, err
				}
				s[i] = nv
			}
			return s, nil
		}
		return v, nil
	}

	for k, v := range with {
		nv, err := walk(v)
		if err != nil {
			return nil, err
		}
		out[k] = nv
	}
	return out, nil
}

func looksLikeTemplate(s string) bool {
	return bytes.Contains([]byte(s), []byte("{{"))
}
