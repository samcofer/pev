# pev — Posit Environment Validator

Assess Linux readiness before installing **Posit Workbench**, **Posit Connect**, and **Posit Package Manager**. One static binary, no runtime dependencies on the target system, runs as root or non-root, writes a Markdown report and a JSON sidecar that diffs cleanly between runs.

[![ci](https://github.com/samcofer/pev/actions/workflows/ci.yml/badge.svg)](https://github.com/samcofer/pev/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/samcofer/pev)](https://github.com/samcofer/pev/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

---

## Why pev exists

Today, the prereq verification call before a Posit install is a manual checklist exercise driven by a runbook, interpreted live by an SE on a Webex call. That process is slow, inconsistent across SEs, and produces no shareable artifact. pev replaces the manual pass with an automated assessment that defaults to discovery (it shells out to the same OS commands a Linux admin would type at the terminal) and produces a graded report identifying every blocking issue *before* the install session — turning the prereq meeting from "let me poke around your box" into "let's review the report you ran yesterday."

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

## Executive summary
| Severity | Pass | Fail | Skip | Unknown |
|---:|---:|---:|---:|---:|
| blocking | 18 | 2 | 1 | 0 |
| warning  | 24 | 3 | 0 | 0 |
| info     | 11 | 0 | 2 | 0 |

**2 blocking failure(s)** — install will not succeed until resolved.
```

## What it checks

Every built-in check maps to an authoritative Posit doc and (where applicable) to a row in the customer prereq runbook. Run `pev list-checks` for the full catalog. Headlines:

| Area | Examples |
|------|----------|
| OS support | Ubuntu 22.04 / 24.04, RHEL/Alma/Rocky 8 / 9 / 10 (Ubuntu 20.04 flagged blocking) |
| Sizing | Workbench 4c/8GB/100GB · Connect 4GB/100GB · PPM 4c/16GB/500GB |
| Network egress | cdn.rstudio.com, cdn.posit.co, packagemanager.posit.co, rspm-sync.rstudio.com, wyDay license activation |
| System packages | gdebi-core, libssl-dev / openssl-devel, libxml2-dev, libcurl-dev |
| Workbench prereqs | customer-supplied SSL cert/key validity, /opt/R/\*, /opt/python/\*, Quarto, IdP metadata reachability |
| Connect prereqs | customer-supplied SSL cert/key validity, SMTP reachability, Quarto |
| PPM prereqs | customer-supplied SSL cert/key validity, rspm-sync.rstudio.com reachable |

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
