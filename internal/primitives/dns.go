package primitives

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/posit-dev/pev/internal/checks"
)

func init() {
	checks.Register("dns", runDNS, []string{
		"name", "type", "must_resolve", "reverse_match_hostname", "timeout_seconds",
	})
}

// runDNS resolves `name` (default A) and optionally checks that a reverse
// lookup of one of its IPs round-trips to `name`.
func runDNS(rc checks.RunCtx) checks.Result {
	name, ok := getString(rc.Check.With, "name")
	if !ok || name == "" {
		return unknownf(rc.Check, "missing required `name` field")
	}
	mustResolve := true
	if v, ok := getBool(rc.Check.With, "must_resolve"); ok {
		mustResolve = v
	}
	timeout := 5 * time.Second
	if t, ok := getInt(rc.Check.With, "timeout_seconds"); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	ctx, cancel := context.WithTimeout(rc.Ctx, timeout)
	defer cancel()
	rc.CmdLog.Append(fmt.Sprintf("nslookup %s", name))

	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title, Severity: rc.Check.Severity,
	}
	addrs, err := net.DefaultResolver.LookupHost(ctx, name)
	if err != nil {
		if !mustResolve {
			r.Status = checks.StatusPass
			r.Reason = "name does not resolve (must_resolve=false)"
			return r
		}
		r.Status = checks.StatusFail
		r.Reason = "lookup: " + err.Error()
		return r
	}
	r.Evidence = []checks.Evidence{{Note: fmt.Sprintf("%s -> %s", name, strings.Join(addrs, ","))}}

	if v, ok := getBool(rc.Check.With, "reverse_match_hostname"); ok && v {
		matched := false
		for _, ip := range addrs {
			names, err := net.DefaultResolver.LookupAddr(ctx, ip)
			if err != nil {
				continue
			}
			for _, n := range names {
				if strings.EqualFold(strings.TrimSuffix(n, "."), name) {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			r.Status = checks.StatusFail
			r.Reason = "no reverse-DNS record for " + name + " matched its forward IP"
			return r
		}
	}

	r.Status = checks.StatusPass
	return r
}
