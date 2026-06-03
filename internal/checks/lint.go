package checks

import (
	"fmt"
	"strings"
)

// Lint validates a list of Checks for catalog correctness: required fields,
// known primitive, allowed `with:` keys per primitive, valid severity, no
// duplicate IDs. Returns one error per violation; callers may wrap them.
func Lint(checks []Check) []error {
	var errs []error
	seen := map[string]string{}

	for _, c := range checks {
		prefix := fmt.Sprintf("[%s in %s]", c.ID, c.Source)

		if c.ID == "" {
			errs = append(errs, fmt.Errorf("%s missing id", prefix))
		}
		if c.Title == "" {
			errs = append(errs, fmt.Errorf("%s missing title", prefix))
		}
		if c.Why == "" {
			errs = append(errs, fmt.Errorf("%s missing why (rationale shown to users)", prefix))
		}
		switch c.Severity {
		case SeverityBlocking, SeverityWarning, SeverityInfo:
		default:
			errs = append(errs, fmt.Errorf("%s invalid severity %q (want blocking|warning|info)", prefix, c.Severity))
		}
		if prev, dup := seen[c.ID]; dup {
			errs = append(errs, fmt.Errorf("%s duplicate id (also in %s)", prefix, prev))
		} else if c.ID != "" {
			seen[c.ID] = c.Source
		}
		if c.Primitive == "" {
			errs = append(errs, fmt.Errorf("%s missing primitive", prefix))
			continue
		}
		if _, err := Lookup(c.Primitive); err != nil {
			errs = append(errs, fmt.Errorf("%s %w", prefix, err))
			continue
		}
		if allowed, ok := AllowedKeys(c.Primitive); ok && len(allowed) > 0 {
			for k := range c.With {
				if _, ok := allowed[k]; !ok {
					errs = append(errs, fmt.Errorf("%s primitive %q does not accept key %q (allowed: %s)",
						prefix, c.Primitive, k, sortedKeys(allowed)))
				}
			}
		}
	}
	return errs
}

func sortedKeys(m map[string]struct{}) string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Cheap deterministic sort without importing sort just for a hot loop.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return strings.Join(out, ",")
}
