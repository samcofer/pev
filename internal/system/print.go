package system

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// PrintAndLogInfo prints to stdout and records the same line via logrus.Info.
// Mirrors wbi's helper; gives every user-visible message an audit-log entry.
func PrintAndLogInfo(msg string) {
	fmt.Println(msg)
	log.Info(msg)
}

// PrintAndLogWarn is the warn-level counterpart.
func PrintAndLogWarn(msg string) {
	fmt.Println(msg)
	log.Warn(msg)
}

// PrintAndLogError is the error-level counterpart. It does NOT abort.
func PrintAndLogError(msg string) {
	fmt.Println(msg)
	log.Error(msg)
}
