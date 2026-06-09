package primitives

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/posit-dev/pev/internal/checks"
)

func init() {
	checks.Register("port", runPort, []string{
		"host", "port", "timeout_seconds",
	})
}

// runPort opens a TCP connection to host:port and reports reachability.
// Equivalent to `nc -vz host port` but built-in.
func runPort(rc checks.RunCtx) checks.Result {
	host, ok := getString(rc.Check.With, "host")
	if !ok || host == "" {
		return unknownf(rc.Check, "missing required `host` field")
	}
	port, ok := getInt(rc.Check.With, "port")
	if !ok || port == 0 {
		return unknownf(rc.Check, "missing required `port` field")
	}
	timeout := 5 * time.Second
	if t, ok := getInt(rc.Check.With, "timeout_seconds"); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title,
		Evidence: []checks.Evidence{{
			Note: fmt.Sprintf("nc -vz %s %d (timeout %s)", host, port, timeout),
		}},
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(rc.Ctx, "tcp", addr)
	if err != nil {
		r.Status = checks.StatusFail
		r.Reason = "tcp dial: " + err.Error()
		return r
	}
	_ = conn.Close()
	r.Status = checks.StatusPass
	return r
}
