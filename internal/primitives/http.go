package primitives

import (
	"fmt"
	"net/http"
	"time"

	"github.com/posit-dev/pev/internal/checks"
)

func init() {
	checks.Register("http", runHTTP, []string{
		"url", "method", "timeout_seconds", "accept_status",
	})
}

// runHTTP issues a GET (or configured method) and accepts the response if its
// status code falls in `accept_status` (default any 2xx).
func runHTTP(rc checks.RunCtx) checks.Result {
	url, ok := getString(rc.Check.With, "url")
	if !ok || url == "" {
		return unknownf(rc.Check, "missing required `url` field")
	}
	method := "GET"
	if m, ok := getString(rc.Check.With, "method"); ok && m != "" {
		method = m
	}
	timeout := 5 * time.Second
	if t, ok := getInt(rc.Check.With, "timeout_seconds"); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}
	accept, _ := getIntSlice(rc.Check.With, "accept_status")

	rc.CmdLog.Append(fmt.Sprintf("curl -I --max-time %d %s", int(timeout.Seconds()), url))

	r := checks.Result{
		ID: rc.Check.ID, Title: rc.Check.Title, Severity: rc.Check.Severity,
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(rc.Ctx, method, url, nil)
	if err != nil {
		return unknownf(rc.Check, "build request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		r.Status = checks.StatusFail
		r.Reason = "request: " + err.Error()
		return r
	}
	defer resp.Body.Close()
	r.Evidence = []checks.Evidence{{Note: fmt.Sprintf("%s %s -> %d", method, url, resp.StatusCode)}}

	if len(accept) > 0 {
		for _, code := range accept {
			if resp.StatusCode == code {
				r.Status = checks.StatusPass
				return r
			}
		}
		r.Status = checks.StatusFail
		r.Reason = fmt.Sprintf("status %d not in accept_status %v", resp.StatusCode, accept)
		return r
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		r.Status = checks.StatusPass
		return r
	}
	r.Status = checks.StatusFail
	r.Reason = fmt.Sprintf("non-2xx status %d", resp.StatusCode)
	return r
}
