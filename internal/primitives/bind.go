package primitives

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/posit-dev/pev/internal/checks"
)

func init() {
	checks.Register("bind", runBind, []string{"host", "port", "timeout_seconds"})
}

// runBind tries to open a TCP listener on host:port. PASS means the host
// can bind there; FAIL means another process is already listening, the
// port is below 1024 and we don't have CAP_NET_BIND_SERVICE, or selinux
// is denying the bind. host defaults to 0.0.0.0 (matches what the Posit
// products do at install). timeout_seconds is currently advisory — the
// listener is opened and immediately closed, so a hang is impossible —
// but kept for future symmetry with the port primitive.
func runBind(rc checks.RunCtx) checks.Result {
	port, ok := getInt(rc.Check.With, "port")
	if !ok || port == 0 {
		return unknownf(rc.Check, "missing required `port` field")
	}
	host := "0.0.0.0"
	if h, ok := getString(rc.Check.With, "host"); ok && h != "" {
		host = h
	}
	timeout := 2 * time.Second
	if t, ok := getInt(rc.Check.With, "timeout_seconds"); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title,
		Evidence: []checks.Evidence{{
			Note: fmt.Sprintf("attempt: net.Listen(\"tcp\", %q) timeout=%s", addr, timeout),
		}},
	}

	var lc net.ListenConfig
	deadline, cancel := context.WithTimeout(rc.Ctx, timeout)
	defer cancel()
	l, err := lc.Listen(deadline, "tcp", addr)
	if err != nil {
		r.Status = checks.StatusFail
		r.Reason = "bind failed: " + err.Error()
		return r
	}
	_ = l.Close()
	r.Status = checks.StatusPass
	r.Reason = "bound and released cleanly"
	return r
}
