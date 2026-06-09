package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/posit-dev/pev/internal/checks"
)

// ANSI colour helpers. Disabled when the writer doesn't support a TTY (we
// don't sniff the terminal here — assess writes to os.Stdout and pipelines
// can flip NO_COLOR or pass --no-color in a future flag iteration).
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

// canonicalCategoryOrder is the order categories appear in the terminal
// output. Categories not in this list sort alphabetically after the known
// set so the new YAML packs naturally append.
var canonicalCategoryOrder = []string{
	checks.CategoryNetworking,
	checks.CategoryStorage,
	checks.CategoryOperatingSystem,
	checks.CategorySecurity,
	checks.CategoryIdentity,
	checks.CategorySSL,
	checks.CategoryPackages,
	checks.CategorySizing,
	checks.CategoryProduct,
	checks.CategoryOther,
}

// RenderTerminal writes the human-facing summary view to w. Shows the
// totals table, then per-category sections that include only failing or
// unknown checks (PASS/SKIP are still in the on-disk Markdown). Output
// is parseable: each finding line is `[STATUS]\tID\tTITLE`.
func RenderTerminal(w io.Writer, rep checks.Report, color bool) {
	red := func(s string) string { return wrap(s, ansiRed, color) }
	yellow := func(s string) string { return wrap(s, ansiYellow, color) }
	bold := func(s string) string { return wrap(s, ansiBold, color) }
	dim := func(s string) string { return wrap(s, ansiDim, color) }

	fmt.Fprintf(w, "%s\n", bold(fmt.Sprintf("pev report — %s — %s",
		rep.Host.Hostname, rep.StartedAt.UTC().Format("2006-01-02 15:04:05 UTC"))))
	fmt.Fprintf(w, "pev %s · schema %d · %d checks · %s\n\n",
		rep.PevVersion, rep.SchemaVersion, rep.Summary.Total,
		rep.FinishedAt.Sub(rep.StartedAt).Round(1e9))

	// Totals — one line, pipe-separated, easy to scan and to grep.
	totals := fmt.Sprintf("Pass %d  |  Fail %d  |  Skip %d  |  Unknown %d",
		rep.Summary.Pass, rep.Summary.Fail, rep.Summary.Skip, rep.Summary.Unknown)
	fmt.Fprintln(w, totals)
	if rep.Summary.Fail > 0 {
		fmt.Fprintf(w, "%s investigate before proceeding.\n",
			red(fmt.Sprintf("%d failure(s) —", rep.Summary.Fail)))
	} else {
		fmt.Fprintf(w, "%s\n", dim("All checks passed."))
	}
	fmt.Fprintln(w)

	renderTerminalEnvironment(w, rep, bold)

	// Group failing/unknown results by category. Pass and Skip stay out
	// of the terminal — they live in the on-disk Markdown for the full
	// audit trail. UNKNOWN counts as a failure here because it means a
	// primitive could not decide.
	byCat := map[string][]checks.Result{}
	for _, r := range rep.Results {
		if r.Status != checks.StatusFail && r.Status != checks.StatusUnknown {
			continue
		}
		byCat[r.Category] = append(byCat[r.Category], r)
	}
	if len(byCat) == 0 {
		fmt.Fprintln(w, dim("All checks passed (or were skipped)."))
		return
	}

	for _, cat := range categoryOrder(byCat) {
		rs := byCat[cat]
		sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
		fmt.Fprintf(w, "%s (%d failing)\n", bold(cat), len(rs))
		for _, r := range rs {
			tag := "[FAIL]"
			if r.Status == checks.StatusUnknown {
				tag = "[UNKN]"
			}
			line := fmt.Sprintf("  %s\t%s\t%s", tag, r.ID, r.Title)
			fmt.Fprintln(w, red(line))
			if r.Reason != "" {
				fmt.Fprintf(w, "    %s %s\n", yellow("reason:"), r.Reason)
			}
			for _, ev := range r.Evidence {
				if ev.Command != "" {
					fmt.Fprintf(w, "    %s %s\n", dim("command:"), ev.Command)
				}
				if ev.Note != "" {
					fmt.Fprintf(w, "    %s %s\n", dim("note:"), ev.Note)
				}
			}
			if r.Why != "" {
				fmt.Fprintf(w, "    %s %s\n", dim("why:"), oneLine(r.Why))
			}
			if r.Remediation != "" {
				fmt.Fprintf(w, "    %s %s\n", yellow("fix:"), oneLine(r.Remediation))
			}
		}
		fmt.Fprintln(w)
	}
}

// renderTerminalEnvironment prints the Environment block on stdout, the
// same shape as `pev discover`. SEs use it as a quick one-line "what is
// this host" reference before scanning failures. Mirrors the on-disk
// Markdown's Environment section so screenshots and the saved report
// agree.
func renderTerminalEnvironment(w io.Writer, rep checks.Report, bold func(string) string) {
	fmt.Fprintln(w, bold("Environment"))
	fmt.Fprintf(w, "  OS:           %s (%s, family=%s)\n", rep.Host.OSPretty, rep.Host.OS, rep.Host.OSFamily)
	fmt.Fprintf(w, "  Architecture: %s\n", rep.Host.Arch)
	fmt.Fprintf(w, "  CPUs:         %d  ·  Memory: %d MB  ·  Disk(/): %d GB free\n",
		rep.Host.CPUs, rep.Host.MemMB, rep.Host.DiskGB["/"])
	if rep.Host.FQDN != "" && rep.Host.FQDN != rep.Host.Hostname {
		fmt.Fprintf(w, "  Hostname:     %s (FQDN: %s)\n", rep.Host.Hostname, rep.Host.FQDN)
	} else {
		fmt.Fprintf(w, "  Hostname:     %s\n", rep.Host.Hostname)
	}
	fmt.Fprintf(w, "  Running root: %v\n", rep.Host.Root)
	fmt.Fprintf(w, "  Detected:     workbench=%v connect=%v packagemanager=%v\n",
		rep.Host.Products.Workbench, rep.Host.Products.Connect, rep.Host.Products.PackageManager)
	if len(rep.Host.R) > 0 {
		fmt.Fprintf(w, "  R:            %s\n", strings.Join(rep.Host.R, ", "))
	}
	if len(rep.Host.Python) > 0 {
		fmt.Fprintf(w, "  Python:       %s\n", strings.Join(rep.Host.Python, ", "))
	}
	if len(rep.Host.Quarto) > 0 {
		fmt.Fprintf(w, "  Quarto:       %s\n", strings.Join(rep.Host.Quarto, ", "))
	}
	if len(rep.Run.Products) > 0 {
		fmt.Fprintf(w, "  Selected:     %s\n", strings.Join(rep.Run.Products, ", "))
	}
	fmt.Fprintln(w)
}

func wrap(s, code string, on bool) string {
	if !on {
		return s
	}
	return code + s + ansiReset
}

func categoryOrder(present map[string][]checks.Result) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, c := range canonicalCategoryOrder {
		if _, ok := present[c]; ok {
			out = append(out, c)
			seen[c] = true
		}
	}
	extra := []string{}
	for c := range present {
		if !seen[c] {
			extra = append(extra, c)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}
