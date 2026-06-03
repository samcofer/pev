package discover

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/posit-dev/pev/internal/system"
)

// versionRoots are the canonical roots for versioned R/Python/Quarto installs
// that Posit products expect to find. Mirrors wbi/internal/languages/*.go.
var (
	rRoots      = []string{"/opt/R", "/opt/local/R", "/usr/local/lib/R"}
	pythonRoots = []string{"/opt/python", "/opt/Python", "/usr/local/lib/python"}
	quartoRoots = []string{"/opt/quarto"}
	quartoExtra = []string{
		"/usr/lib/rstudio-server/bin/quarto/bin/quarto",
		"/usr/local/bin/quarto",
	}
)

// ScanRVersions returns every directory under known R roots whose <dir>/bin/R exists.
func ScanRVersions() []string { return scanVersioned(rRoots, "bin/R") }

// ScanPythonVersions returns every <dir>/bin/python or python3 under known Python roots.
func ScanPythonVersions() []string {
	out := scanVersioned(pythonRoots, "bin/python3")
	out = append(out, scanVersioned(pythonRoots, "bin/python")...)
	return dedupeSorted(out)
}

// ScanQuartoVersions returns versioned roots plus the bundled+symlink fallbacks.
func ScanQuartoVersions() []string {
	out := scanVersioned(quartoRoots, "bin/quarto")
	for _, p := range quartoExtra {
		if system.FileExists(p) {
			out = append(out, p)
		}
	}
	return dedupeSorted(out)
}

func scanVersioned(roots []string, suffix string) []string {
	out := []string{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			cand := filepath.Join(root, e.Name(), suffix)
			if system.FileExists(cand) {
				out = append(out, cand)
			}
		}
	}
	return dedupeSorted(out)
}

func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
