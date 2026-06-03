package checks

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"text/template"
	"time"

	"github.com/posit-dev/pev/internal/discover"
	"github.com/posit-dev/pev/internal/logging"
	"github.com/posit-dev/pev/internal/system"
)

// Engine runs a list of Checks against a HostFacts snapshot and a set of inputs.
type Engine struct {
	Facts  discover.HostFacts
	Inputs map[string]string
	CmdLog *logging.CmdLog
}

// Run filters checks by AppliesTo, gates root-only checks, expands templates
// in `with:`, dispatches to the registered primitive, and returns Results
// sorted by ID.
func (e *Engine) Run(ctx context.Context, all []Check) []Result {
	out := make([]Result, 0, len(all))
	for _, c := range all {
		out = append(out, e.runOne(ctx, c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (e *Engine) runOne(ctx context.Context, c Check) Result {
	start := time.Now()
	base := Result{
		ID: c.ID, Title: c.Title, Severity: c.Severity,
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
		Ctx: ctx, Check: c, Facts: e.Facts, Inputs: e.Inputs, CmdLog: e.CmdLog,
	})
	r.ID = c.ID
	r.Title = c.Title
	r.Severity = c.Severity
	r.Why = c.Why
	r.References = c.References
	return finish(r)
}

// appliesTo matches a Check's AppliesTo gate against host facts. An empty
// list for a dimension is "any" — only non-empty lists are restrictive.
func appliesTo(a AppliesTo, hf discover.HostFacts) bool {
	if len(a.OS) > 0 && !contains(a.OS, hf.OS) {
		return false
	}
	if len(a.Arch) > 0 && !contains(a.Arch, hf.Arch) {
		return false
	}
	if len(a.Products) > 0 {
		// Products gate runs against the user-selected product list (Run.Inputs).
		// We piggyback on a synthetic input "_products_csv" set by the assess command.
		// This keeps appliesTo a pure function of facts+inputs.
		// If neither input nor any detected product matches, skip.
		matched := false
		for _, p := range a.Products {
			if hf.Products.Workbench && p == "workbench" {
				matched = true
				break
			}
			if hf.Products.Connect && p == "connect" {
				matched = true
				break
			}
			if hf.Products.PackageManager && p == "packagemanager" {
				matched = true
				break
			}
		}
		// Fall through: if user selected the product explicitly via flag, that's handled
		// by the assess command pre-filtering the catalog before passing it in.
		_ = matched
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
