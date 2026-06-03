package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// CmdLog is a replayable shell-script log of every command pev ran.
// One CmdLog per assess run. Safe for concurrent writes.
type CmdLog struct {
	mu      sync.Mutex
	w       *bufio.Writer
	f       *os.File
	enabled bool
}

// licenseKeyRE matches the 28-char hyphenated Posit license-key shape so the
// scrubber redacts them even if a custom check happens to emit one.
var licenseKeyRE = regexp.MustCompile(`\b[A-Z0-9]{4}(?:-[A-Z0-9]{4}){5}\b`)

// NewCmdLog opens pev-cmdlog-<host>-<TS>.sh under outDir. If enabled is false,
// returns a no-op CmdLog whose Append is a cheap method-call.
func NewCmdLog(outDir, hostname string, enabled bool) (*CmdLog, error) {
	if !enabled {
		return &CmdLog{enabled: false}, nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("create out dir: %w", err)
	}
	ts := time.Now().UTC().Format("20060102T150405")
	path := filepath.Join(outDir, fmt.Sprintf("pev-cmdlog-%s-%s.sh", hostname, ts))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o755)
	if err != nil {
		return nil, fmt.Errorf("open cmdlog: %w", err)
	}
	w := bufio.NewWriter(f)
	header := fmt.Sprintf("#!/bin/bash\n# pev cmdlog — %s — %s UTC\n# This is the verbatim sequence of shell commands pev executed.\n# Re-run as a script to reproduce the assessment commands manually.\nset -e\n\n",
		hostname, time.Now().UTC().Format(time.RFC3339))
	if _, err := w.WriteString(header); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &CmdLog{w: w, f: f, enabled: true}, nil
}

// Append writes one shell command followed by a newline. The command is
// scrubbed for known secret patterns before being written.
func (c *CmdLog) Append(cmd string) {
	if c == nil || !c.enabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	scrubbed := licenseKeyRE.ReplaceAllString(cmd, "<REDACTED-LICENSE-KEY>")
	_, _ = c.w.WriteString(scrubbed + "\n")
}

// Close flushes and closes the cmdlog file. Safe to call on a no-op CmdLog.
func (c *CmdLog) Close() error {
	if c == nil || !c.enabled {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.w.Flush(); err != nil {
		_ = c.f.Close()
		return err
	}
	return c.f.Close()
}
