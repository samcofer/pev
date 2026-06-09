# pev — Posit Environment Validator

Assess Linux readiness before installing **Posit Workbench**, **Posit Connect**, and **Posit Package Manager**. One static binary, no runtime dependencies on the target system, runs as root or non-root, writes a Markdown report and a JSON sidecar that diffs cleanly between runs.

[![ci](https://github.com/samcofer/pev/actions/workflows/ci.yml/badge.svg)](https://github.com/samcofer/pev/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/samcofer/pev)](https://github.com/samcofer/pev/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

---

## Why pev exists

Today, the prereq verification call before a Posit install is a manual checklist exercise driven by a runbook, interpreted live by an SE on a Webex call. That process is slow, inconsistent across SEs, and produces no shareable artifact. pev replaces the manual pass with an automated assessment that defaults to discovery (it shells out to the same OS commands a Linux admin would type at the terminal) and produces a report identifying every issue *before* the install session — turning the prereq meeting from "let me poke around your box" into "let's review the report you ran yesterday." Every failed check is worth investigating; pev does not try to predict which failures the customer can defer.

## Scope

**pev assesses the host's readiness BEFORE any Posit product is installed.** It checks OS support, sizing, network egress, system packages, customer-supplied SSL cert/key validity, R/Python/Quarto presence at the expected paths, IdP and SMTP reachability — every prereq the customer has to satisfy ahead of the install session.

**pev explicitly does NOT validate installed products.** License activation (`rstudio-server license-manager status`), product binary presence, post-install config files (`rserver.conf`, `rstudio-connect.gcfg`, `rstudio-pm.gcfg`), and content-deployment smoke tests are the responsibility of [`posit-dev/vip`](https://github.com/posit-dev/vip). If a check requires a Posit product to be installed first, it belongs in `vip`, not here.

## Install

### Verified install (recommended)

Releases are signed with [cosign](https://github.com/sigstore/cosign) keyless OIDC.

```bash
# 1. Download the binary and the signed checksums file.
curl -fsSL https://github.com/samcofer/pev/releases/latest/download/pev_linux_amd64 -o pev
curl -fsSL https://github.com/samcofer/pev/releases/latest/download/pev_VERSION_checksums.txt -o checksums.txt
curl -fsSL https://github.com/samcofer/pev/releases/latest/download/pev_VERSION_checksums.txt.sig -o checksums.txt.sig

# 2. Verify the checksums file's signature against the GitHub Actions OIDC identity.
cosign verify-blob \
  --certificate-identity-regexp 'github.com/samcofer/pev' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature checksums.txt.sig checksums.txt

# 3. Verify the binary against the checksums file.
sha256sum --check --ignore-missing checksums.txt

chmod +x pev
./pev assess
```

### Plain install (air-gapped customers)

```bash
curl -fsSL https://github.com/samcofer/pev/releases/latest/download/pev_linux_amd64 -o pev
chmod +x pev
./pev assess
```

Compare the SHA-256 against the value in the release notes if you skip cosign.

## Quickstart

```bash
./pev discover                       # what facts pev would assume — no checks run
./pev assess                         # Markdown to screen + Markdown + JSON to ./
./pev assess --out-dir /var/log/pev  # land artifacts in a stable location
./pev diff a.json b.json             # exit 1 on regressions
```

A trimmed report excerpt:

```
# pev report — db-prod-01 — 2026-06-03 14:22:05 UTC

## Summary
| Pass | Fail | Skip | Unknown |
|---:|---:|---:|---:|
|   53 |    5 |    3 |    0 |

**5 failure(s)** — investigate before proceeding.
```

## What it checks

Every built-in check maps to an authoritative Posit doc and (where applicable) to a row in the customer prereq runbook. Run `pev list-checks` to dump the catalog at any time.

The full built-in catalog (run `pev list-checks` for the live version):

### Operating System

| ID | Title |
|---|---|
| `os.supported` | Operating system is supported by Posit professional products |
| `os.architecture.amd64-or-arm64` | CPU architecture is amd64 or arm64 |
| `os.tmp.exec` | /tmp is mounted exec (not noexec) |
| `os.home.exec` | /home is mounted exec (not noexec) |

### Sizing

| ID | Title |
|---|---|
| `sizing.workbench.minimum` | Host meets Workbench minimum sizing (4 cores / 8 GB / 100 GB) |
| `sizing.connect.minimum` | Host meets Connect minimum sizing (4 GB / 100 GB) |
| `sizing.packagemanager.recommended` | Host meets Package Manager recommended sizing (4 cores / 16 GB / 500 GB) |

### Networking — Egress

| ID | Title |
|---|---|
| `net.egress.cdn-rstudio` | TLS reachability to cdn.rstudio.com:443 |
| `net.egress.cdn-posit` | TLS reachability to cdn.posit.co:443 |
| `net.egress.download2-rstudio` | TLS reachability to download2.rstudio.org:443 |
| `net.egress.license-activation` | TLS reachability to www.wyday.com:443 (license activation) |
| `net.egress.packagemanager-posit-ping` | HTTPS GET packagemanager.posit.co/__ping__ returns 200 |
| `net.egress.p3m` | TLS reachability to p3m.dev:443 (Posit Public Package Manager) |
| `net.egress.cran` | TLS reachability to cran.r-project.org:443 |
| `net.egress.bioconductor` | TLS reachability to bioconductor.org:443 |
| `net.egress.pypi` | TLS reachability to pypi.org:443 |
| `net.egress.pypi-files` | TLS reachability to files.pythonhosted.org:443 |
| `ppm.egress.sync` | Package Manager can reach the Posit Package Service |

### Security

| ID | Title |
|---|---|
| `sec.selinux.status` | SELinux mode is reported (RHEL family) |
| `sec.apparmor.status` | AppArmor mode is reported (Ubuntu) |
| `sec.firewalld.inactive` | firewalld is not active (or rules permit Posit ports) |
| `sec.iptables.inactive` | iptables service is not active (or rules permit Posit ports) |
| `sec.nftables.inactive` | nftables service is not active (or rules permit Posit ports) |

### Distro Package Manager Health

| ID | Title |
|---|---|
| `pkg-mgr.apt.update` | apt-get update succeeds (Ubuntu) |
| `pkg-mgr.apt.repolist-fresh` | apt repository metadata is recent (< 30 days) |
| `pkg-mgr.dnf.repolist` | dnf repolist succeeds (RHEL) |
| `pkg-mgr.dnf.makecache` | dnf makecache succeeds |

### Build-Dep System Packages

| ID | Title |
|---|---|
| `pkg.gdebi.ubuntu` | gdebi-core installed (Ubuntu) |
| `pkg.openssl-dev` | openssl development headers installed |
| `pkg.libcurl-dev` | libcurl development headers installed |
| `pkg.libxml2-dev` | libxml2 development headers installed |

### SSL / TLS (customer-supplied; opt-in prompt)

| ID | Title |
|---|---|
| `workbench.ssl.cert-key-match` | Workbench SSL certificate and key are paired |
| `connect.ssl.cert-key-match` | Connect SSL certificate and key are paired |
| `ppm.ssl.cert-key-match` | Package Manager SSL certificate and key are paired |

### Languages & Identity (common to all products)

These checks are no longer scoped to a single product — they apply to any
host running Workbench, Connect, or Package Manager. The user-install
checks run as the unprivileged user pev auto-detects (or prompts for, if
running as root) and use the latest discovered R / Python under `/opt`.

| ID | Title |
|---|---|
| `lang.r.versioned-install` | At least one R install at `/opt/R/<version>/bin/R` |
| `lang.r.renv-user-install` | Unprivileged user can install renv with the latest R |
| `lang.python.versioned-install` | At least one Python install at `/opt/python/<version>/bin/python3` |
| `lang.python.uv-venv` | Unprivileged user can create a uv venv with the latest Python |
| `lang.python.pip-venv` | Unprivileged user can create a venv via `python -m venv` + pip install |
| `lang.quarto.present` | Quarto is available on PATH |
| `lang.idp.metadata` | IdP metadata or discovery URL is reachable |

### Connect-specific

| ID | Title |
|---|---|
| `connect.smtp.reachable` | SMTP server reachable from Connect host |

Anything that requires a Posit product to be already installed (license-manager status, parsing rserver.conf, etc.) is **out of scope** — that's `vip`'s job. See [docs/runbook-mapping.md](docs/runbook-mapping.md) for the full prereq → check ID table and the explicit out-of-scope list.

## Permissions

pev runs as root or non-root. The pre-install catalog is mostly readable by any user (DNS, HTTP, /opt/* listings, sysctl). Checks that need root (e.g. reading customer-supplied SSL keys at mode 0600) are gated by `requires_root: true` and emit `SKIPPED (requires root)` when run as a normal user. The run never aborts.

```bash
sudo ./pev assess         # full coverage, including SSL-key checks
./pev assess              # everything that doesn't need root
```

## Custom checks

Drop a YAML file in `~/.config/pev/checks.d/` (loaded automatically) or pass `--checks-file path.yaml`:

```yaml
schema_version: 1
checks:
  - id: mycorp.cron.installed
    title: Internal cron job for log rotation is installed
    severity: warning
    tags: [internal]
    primitive: file
    why: |
      Our prod boxes ship a custom logrotate.d unit. If it's missing, /var fills up.
    with:
      path: /etc/logrotate.d/posit-mycorp
      must_exist: true
```

`pev lint-checks file.yaml` validates a pack before you ship it. See [docs/check-authoring.md](docs/check-authoring.md) for the full schema and [docs/primitives.md](docs/primitives.md) for the available primitives.

## Reports & diffs

Every `pev assess` writes four files:

- `pev-report-<host>-<TS>.md` — human Markdown report
- `pev-report-<host>-<TS>.json` — machine sidecar (stable, sorted, schema-versioned)
- `pev-cmdlog-<host>-<TS>.sh` — replayable shell script of every command pev ran (license keys redacted)
- `pev-log-<TS>.log` — logrus JSON-lines for debugging

`pev diff a.json b.json` classifies every check as **regression** (PASS→FAIL/UNKNOWN), **improvement** (FAIL→PASS), **status changed**, **added**, **removed**, or **evidence-only changed**. Exit code 1 iff regressions exist — perfect for a CI gate during install runbook automation.

## Supported OS

| OS                       | Status                  | Notes                                                |
|--------------------------|-------------------------|------------------------------------------------------|
| Ubuntu 22.04             | Supported               |                                                       |
| Ubuntu 24.04             | Supported               |                                                       |
| Ubuntu 20.04             | **Unsupported** (block) | EOL across all three Posit products                   |
| RHEL 8 / 9 / 10          | Supported               | UBI requires registry auth                            |
| Alma Linux 8 / 9 / 10    | Supported               | RHEL-family rebuild; collapsed onto `rhel-<major>` ID |
| Rocky Linux 8 / 9 / 10   | Supported               | Same as Alma                                          |

CI exercises Ubuntu 22.04, Ubuntu 24.04, Alma 9, and Alma 10 containers. Real RHEL is validated pre-release on customer-representative VMs.

## Building from source

```bash
git clone https://github.com/samcofer/pev
cd pev
make build       # CGO_ENABLED=0 -> ~12 MB static binary
make test        # go test ./... -race -shuffle=on
make lint        # golangci-lint
make snapshot    # goreleaser release --snapshot --clean
make e2e         # local docker matrix (Ubuntu 22/24, Alma 9/10)
```

Requires Go 1.22+.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`); the changelog is generated from those prefixes.

## Security

See [SECURITY.md](SECURITY.md). Verify release artifacts with cosign as shown above. Report vulnerabilities privately via GitHub Security Advisories.

## Maintainers

Posit Solutions Engineering. Internal Slack: `#se-tools`. For customer-facing escalations, file via the standard Solutions support flow.
