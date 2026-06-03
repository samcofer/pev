package discover

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// readCPUs returns logical CPU count via runtime, with /proc/cpuinfo as fallback.
func readCPUs() int {
	if n := runtime.NumCPU(); n > 0 {
		return n
	}
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	count := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.HasPrefix(s.Text(), "processor") {
			count++
		}
	}
	return count
}

// readMemMB returns total memory in MB by parsing /proc/meminfo.
func readMemMB() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0
		}
		return kb / 1024
	}
	return 0
}

// readDiskFreeGB returns free disk on the filesystem containing path, in GB.
func readDiskFreeGB(path string) int {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0
	}
	// Bavail * Bsize = bytes available to non-root; use this for honest "free" reporting.
	bytes := uint64(st.Bavail) * uint64(st.Bsize)
	return int(bytes / (1024 * 1024 * 1024))
}
