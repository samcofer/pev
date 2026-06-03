package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/posit-dev/pev/internal/checks"
)

// WriteMarkdown writes the report as Markdown to outDir/<base>.md.
func WriteMarkdown(outDir, base string, rep checks.Report) (string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(outDir, base+".md")
	if err := os.WriteFile(path, []byte(RenderMarkdown(rep)), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// RenderMarkdown produces the Markdown body for a report. Deterministic:
// results sorted by ID, sections in fixed order, no timestamps in section
// titles to keep diffs clean.
func RenderMarkdown(rep checks.Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# pev report — %s — %s\n\n", rep.Host.Hostname, rep.StartedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(&b, "**pev** %s · schema %d · duration %s\n\n",
		rep.PevVersion, rep.SchemaVersion, rep.FinishedAt.Sub(rep.StartedAt).Round(1e9))

	// Executive summary
	b.WriteString("## Executive summary\n\n")
	bySev := map[checks.Severity]map[checks.Status]int{}
	for _, r := range rep.Results {
		if bySev[r.Severity] == nil {
			bySev[r.Severity] = map[checks.Status]int{}
		}
		bySev[r.Severity][r.Status]++
	}
	b.WriteString("| Severity | Pass | Fail | Skip | Unknown |\n")
	b.WriteString("|---:|---:|---:|---:|---:|\n")
	for _, sev := range []checks.Severity{checks.SeverityBlocking, checks.SeverityWarning, checks.SeverityInfo} {
		row := bySev[sev]
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n",
			sev, row[checks.StatusPass], row[checks.StatusFail], row[checks.StatusSkip], row[checks.StatusUnknown])
	}
	if rep.Summary.Blocking > 0 {
		fmt.Fprintf(&b, "\n**%d blocking failure(s)** — install will not succeed until resolved.\n\n", rep.Summary.Blocking)
	} else {
		b.WriteString("\nNo blocking failures.\n\n")
	}

	// Environment
	b.WriteString("## Environment\n\n")
	fmt.Fprintf(&b, "- OS: %s (%s, family=%s)\n", rep.Host.OSPretty, rep.Host.OS, rep.Host.OSFamily)
	fmt.Fprintf(&b, "- Architecture: %s\n", rep.Host.Arch)
	fmt.Fprintf(&b, "- CPUs: %d · Memory: %d MB · Disk(/): %d GB free\n", rep.Host.CPUs, rep.Host.MemMB, rep.Host.DiskGB["/"])
	fmt.Fprintf(&b, "- Hostname: %s", rep.Host.Hostname)
	if rep.Host.FQDN != "" && rep.Host.FQDN != rep.Host.Hostname {
		fmt.Fprintf(&b, " (FQDN: %s)", rep.Host.FQDN)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "- Running as root: %v\n", rep.Host.Root)
	fmt.Fprintf(&b, "- Detected products: workbench=%v connect=%v packagemanager=%v\n",
		rep.Host.Products.Workbench, rep.Host.Products.Connect, rep.Host.Products.PackageManager)
	if len(rep.Host.R) > 0 {
		fmt.Fprintf(&b, "- R installs: %s\n", strings.Join(rep.Host.R, ", "))
	}
	if len(rep.Host.Python) > 0 {
		fmt.Fprintf(&b, "- Python installs: %s\n", strings.Join(rep.Host.Python, ", "))
	}
	if len(rep.Host.Quarto) > 0 {
		fmt.Fprintf(&b, "- Quarto installs: %s\n", strings.Join(rep.Host.Quarto, ", "))
	}
	if len(rep.Run.Products) > 0 {
		fmt.Fprintf(&b, "- Selected products: %s\n", strings.Join(rep.Run.Products, ", "))
	}
	if rep.Run.Profile != "" {
		fmt.Fprintf(&b, "- Profile: %s\n", rep.Run.Profile)
	}
	b.WriteString("\n")

	// Findings
	b.WriteString("## Findings\n\n")
	groups := groupBy(rep.Results)
	for _, sev := range []checks.Severity{checks.SeverityBlocking, checks.SeverityWarning, checks.SeverityInfo} {
		rs := groups[sev]
		if len(rs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s (%d)\n\n", titleSeverity(sev), len(rs))
		for _, r := range rs {
			icon := iconFor(r.Status)
			fmt.Fprintf(&b, "- %s **%s** `%s` — %s\n", icon, strings.ToUpper(string(r.Status)), r.ID, r.Title)
			if r.Reason != "" {
				fmt.Fprintf(&b, "  - Reason: %s\n", r.Reason)
			}
			if r.Why != "" {
				fmt.Fprintf(&b, "  - Why: %s\n", oneLine(r.Why))
			}
			for _, ev := range r.Evidence {
				if ev.Command != "" {
					fmt.Fprintf(&b, "  - Command: `%s`\n", ev.Command)
				}
				if ev.Path != "" {
					fmt.Fprintf(&b, "  - Path: `%s`\n", ev.Path)
				}
				if ev.Note != "" {
					fmt.Fprintf(&b, "  - Note: %s\n", ev.Note)
				}
			}
			for _, ref := range r.References {
				fmt.Fprintf(&b, "  - Reference: %s\n", ref)
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func groupBy(in []checks.Result) map[checks.Severity][]checks.Result {
	out := map[checks.Severity][]checks.Result{}
	for _, r := range in {
		out[r.Severity] = append(out[r.Severity], r)
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].ID < out[k][j].ID })
	}
	return out
}

func iconFor(s checks.Status) string {
	switch s {
	case checks.StatusPass:
		return "[PASS]"
	case checks.StatusFail:
		return "[FAIL]"
	case checks.StatusSkip:
		return "[SKIP]"
	default:
		return "[????]"
	}
}

func titleSeverity(s checks.Severity) string {
	switch s {
	case checks.SeverityBlocking:
		return "Blocking"
	case checks.SeverityWarning:
		return "Warning"
	case checks.SeverityInfo:
		return "Info"
	}
	return string(s)
}

func oneLine(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
