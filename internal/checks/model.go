// Package checks defines the data model, registry, loader, and engine for
// pev's check catalog. Checks themselves live in YAML packs under /checks
// (embedded at build time) and may be supplemented by user-supplied YAML.
package checks

import (
	"time"

	"github.com/posit-dev/pev/internal/discover"
)

// SchemaVersion is the major version of the pev report and YAML schemas.
// Bumped only on breaking changes; pev diff rejects mismatched majors.
const SchemaVersion = 1

// Severity grades how badly a failed check affects an install.
type Severity string

const (
	SeverityBlocking Severity = "blocking"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

// Status is the outcome of running a check.
type Status string

const (
	StatusPass    Status = "pass"
	StatusFail    Status = "fail"
	StatusSkip    Status = "skip"
	StatusUnknown Status = "unknown"
)

// AppliesTo gates a check by host facts.
type AppliesTo struct {
	OS       []string `yaml:"os" json:"os,omitempty"`
	Products []string `yaml:"products" json:"products,omitempty"`
	Roles    []string `yaml:"roles" json:"roles,omitempty"`
	Arch     []string `yaml:"arch" json:"arch,omitempty"`
}

// Check is one entry in the catalog.
type Check struct {
	ID           string                 `yaml:"id" json:"id"`
	Title        string                 `yaml:"title" json:"title"`
	Severity     Severity               `yaml:"severity" json:"severity"`
	Tags         []string               `yaml:"tags" json:"tags,omitempty"`
	AppliesTo    AppliesTo              `yaml:"applies_to" json:"applies_to,omitempty"`
	RequiresRoot bool                   `yaml:"requires_root" json:"requires_root,omitempty"`
	Why          string                 `yaml:"why" json:"why,omitempty"`
	Remediation  string                 `yaml:"remediation" json:"remediation,omitempty"`
	References   []string               `yaml:"references" json:"references,omitempty"`
	Primitive    string                 `yaml:"primitive" json:"primitive"`
	With         map[string]interface{} `yaml:"with" json:"with,omitempty"`
	Source       string                 `yaml:"-" json:"-"` // pack origin, for diagnostics
}

// Pack is one YAML file's contents.
type Pack struct {
	SchemaVersion int     `yaml:"schema_version"`
	Checks        []Check `yaml:"checks"`
}

// Evidence is a single piece of supporting output for a Result.
type Evidence struct {
	Command string `json:"command,omitempty"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Path    string `json:"path,omitempty"`
	Note    string `json:"note,omitempty"`
}

// Result is what one check produced.
type Result struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Severity   Severity   `json:"severity"`
	Status     Status     `json:"status"`
	Why        string     `json:"why,omitempty"`
	Reason     string     `json:"reason,omitempty"` // short pass/fail/skip explanation
	Evidence   []Evidence `json:"evidence,omitempty"`
	References []string   `json:"references,omitempty"`
	DurationMS int64      `json:"duration_ms"`
}

// Summary tallies a Report's outcomes.
type Summary struct {
	Total    int `json:"total"`
	Pass     int `json:"pass"`
	Fail     int `json:"fail"`
	Skip     int `json:"skip"`
	Unknown  int `json:"unknown"`
	Blocking int `json:"blocking_failures"`
}

// Run records the assessment's input parameters.
type Run struct {
	Products []string          `json:"products"`
	Profile  string            `json:"profile,omitempty"`
	Inputs   map[string]string `json:"inputs,omitempty"`
}

// Report is the on-disk artifact emitted by `pev assess` (and rendered to Markdown).
type Report struct {
	PevVersion    string             `json:"pev_version"`
	SchemaVersion int                `json:"schema_version"`
	Host          discover.HostFacts `json:"host"`
	Run           Run                `json:"run"`
	StartedAt     time.Time          `json:"started_at"`
	FinishedAt    time.Time          `json:"finished_at"`
	Summary       Summary            `json:"summary"`
	Results       []Result           `json:"results"`
}
