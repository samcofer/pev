// Package discover performs read-only probes that build up the HostFacts
// snapshot used to gate checks and seed prompts.
package discover

import (
	"context"
	"os"
	"runtime"

	"github.com/posit-dev/pev/internal/operatingsystem"
	"github.com/posit-dev/pev/internal/system"
)

// HostFacts is the structured snapshot of the host pev assesses.
type HostFacts struct {
	Hostname string         `json:"hostname"`
	FQDN     string         `json:"fqdn,omitempty"`
	OS       string         `json:"os"`        // canonical id, e.g. ubuntu-22.04
	OSPretty string         `json:"os_pretty"` // human pretty name
	OSFamily string         `json:"os_family"` // ubuntu | rhel | unknown
	Arch     string         `json:"arch"`      // amd64 | arm64
	CPUs     int            `json:"cpus"`
	MemMB    int            `json:"mem_mb"`
	DiskGB   map[string]int `json:"disk_gb"` // mountpoint -> free GB
	Root     bool           `json:"root"`
	Products Products       `json:"products"`
	R        []string       `json:"r_versions,omitempty"`
	Python   []string       `json:"python_versions,omitempty"`
	Quarto   []string       `json:"quarto_versions,omitempty"`
}

// Gather runs every discovery probe and returns the resulting HostFacts.
// Probes that fail emit a warning to the log but never abort.
func Gather(ctx context.Context) HostFacts {
	osd := operatingsystem.Detect()
	hostname, _ := os.Hostname()

	hf := HostFacts{
		Hostname: hostname,
		FQDN:     resolveFQDN(ctx, hostname),
		OS:       osd.ID,
		OSPretty: osd.Pretty,
		OSFamily: osd.Family,
		Arch:     runtime.GOARCH,
		Root:     system.IsRoot(),
		DiskGB:   map[string]int{},
	}

	hf.CPUs = readCPUs()
	hf.MemMB = readMemMB()
	hf.DiskGB["/"] = readDiskFreeGB("/")
	hf.Products = DetectProducts()
	hf.R = ScanRVersions()
	hf.Python = ScanPythonVersions()
	hf.Quarto = ScanQuartoVersions()
	return hf
}
