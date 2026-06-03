package primitives

import (
	"fmt"
	"strings"

	"github.com/posit-dev/pev/internal/checks"
)

func init() {
	checks.Register("sizing", runSizing, []string{"cpus_min", "mem_gb_min", "disk_gb_min"})
}

// runSizing compares discovered host facts against thresholds. No shell-outs;
// values come from HostFacts populated during discovery.
func runSizing(rc checks.RunCtx) checks.Result {
	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title, Severity: rc.Check.Severity,
	}
	var failures []string

	if min, ok := getInt(rc.Check.With, "cpus_min"); ok && rc.Facts.CPUs < min {
		failures = append(failures, fmt.Sprintf("cpus=%d<min=%d", rc.Facts.CPUs, min))
	}
	if min, ok := getInt(rc.Check.With, "mem_gb_min"); ok {
		gotGB := rc.Facts.MemMB / 1024
		if gotGB < min {
			failures = append(failures, fmt.Sprintf("mem_gb=%d<min=%d", gotGB, min))
		}
	}
	if disk, ok := rc.Check.With["disk_gb_min"].(map[string]interface{}); ok {
		for mp, raw := range disk {
			min := 0
			switch v := raw.(type) {
			case int:
				min = v
			case int64:
				min = int(v)
			case float64:
				min = int(v)
			}
			got := rc.Facts.DiskGB[mp]
			if got < min {
				failures = append(failures, fmt.Sprintf("disk_gb[%s]=%d<min=%d", mp, got, min))
			}
		}
	}

	r.Evidence = []checks.Evidence{{
		Note: fmt.Sprintf("cpus=%d mem_mb=%d disk_gb=%v", rc.Facts.CPUs, rc.Facts.MemMB, rc.Facts.DiskGB),
	}}
	if len(failures) > 0 {
		r.Status = checks.StatusFail
		r.Reason = strings.Join(failures, "; ")
		return r
	}
	r.Status = checks.StatusPass
	return r
}
