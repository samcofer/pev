# Primitives reference

A check's `primitive:` field selects which executor runs the check. v1 ships eleven primitives.

## `cmd`

Run a shell command via `/bin/sh -c` and match its output.

```yaml
primitive: cmd
with:
  cmd: "rstudio-server license-manager status"
  expect_exit: 0
  expect_stdout_regex: 'Status:\s+Activated'
  expect_stderr_regex: ''             # optional
  timeout_seconds: 30
```

| Key | Default | Notes |
|---|---|---|
| `cmd` | — | required |
| `expect_exit` | unset | when set, `exit code != value` ⇒ fail |
| `expect_stdout_regex` | unset | Go regex; mismatch ⇒ fail |
| `expect_stderr_regex` | unset | Go regex; mismatch ⇒ fail |
| `timeout_seconds` | 30 | command is killed and check fails |

## `file`

Single-file inspection.

```yaml
primitive: file
with:
  path: /etc/rstudio/rserver.conf
  must_exist: true
  mode_max: "0644"            # fail if mode is more permissive
  contains_regex: "(?m)^ssl-enabled=1"
```

## `dir`

Directory presence and optional glob match count.

```yaml
primitive: dir
with:
  path: /var/lib/rstudio-server
  must_exist: true
  glob: "*.lic"
  glob_min_matches: 1
```

## `port`

Built-in `nc -vz`. Opens a TCP connection.

```yaml
primitive: port
with:
  host: "{{ .Inputs.connect_smtp_host }}"
  port: 587
  timeout_seconds: 5
```

## `dns`

Forward resolution with an optional reverse-match check.

```yaml
primitive: dns
with:
  name: "{{ .Inputs.workbench_hostname }}"
  type: A
  must_resolve: true
  reverse_match_hostname: false
  timeout_seconds: 5
```

## `http`

GET (or configured method) and accept any 2xx by default.

```yaml
primitive: http
with:
  url: https://cdn.posit.co/
  method: GET
  timeout_seconds: 5
  accept_status: [200, 204, 301, 302]   # optional; otherwise any 2xx
```

## `x509`

Validate a PEM certificate: chain against the system trust store, hostname match, expiration cushion, and RSA cert↔key pairing via modulus comparison (no openssl shell-out needed).

```yaml
primitive: x509
with:
  cert_path: "{{ .Inputs.workbench_cert }}"
  key_path:  "{{ .Inputs.workbench_key }}"
  verify_chain: true
  match_hostname: "{{ .Inputs.workbench_hostname }}"
  not_after_min_days: 30
```

EC private keys are not supported in v1 — file an issue if you need one.

## `pkg`

Distro package presence via `dpkg-query` (Ubuntu) or `rpm` (RHEL family).

```yaml
primitive: pkg
with:
  manager: auto                # auto | dpkg | rpm
  any_of: [libssl-dev, openssl-devel]   # any present ⇒ pass
  # all_of: [pkg1, pkg2]                # all present ⇒ pass
```

`auto` picks `dpkg` on Ubuntu and `rpm` on RHEL family. Provide both Ubuntu and RHEL package names in `any_of` for cross-distro checks.

## `proc`

Systemd unit state.

```yaml
primitive: proc
with:
  unit: rstudio-server
  state: active        # any value `systemctl is-<state>` accepts
```

## `sysctl`

Read a kernel parameter from `/proc/sys/<dotted/path>`.

```yaml
primitive: sysctl
with:
  key: net.ipv4.tcp_keepalive_time
  expect_int_min: 60
  # expect_equals: "1"
```

## `sizing`

Check host capacity against thresholds. Reads from `HostFacts` (no shell-out).

```yaml
primitive: sizing
with:
  cpus_min: 4
  mem_gb_min: 8
  disk_gb_min: { /: 100, /var: 50 }
```

Disk values are *available* GB to non-root (matches `df -h`'s Available column).
