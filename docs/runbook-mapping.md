# Runbook prereq → check ID mapping

Each row in the customer onboarding runbook (the BCBS-style "Prerequisites" tab) maps to one or more pev built-in checks. **pev's scope is strictly pre-install prerequisites.** Validation of an *installed* Posit product (license activation, rserver.conf SSL config, deployed-content tests) is the responsibility of [`vip`](https://github.com/posit-dev/vip), not pev.

This file is the contract — when a runbook prereq lands here, the corresponding YAML check exists in the catalog and is exercised by CI.

## Shared infrastructure (all products)

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| Linux VM provisioned (CPU/RAM/disk) | `sizing.workbench.minimum`, `sizing.connect.minimum`, `sizing.packagemanager.recommended` | warning |
| OS supported | `os.supported` | blocking |
| CPU architecture | `os.architecture.amd64-or-arm64` | blocking |
| Outbound networking — Posit CDNs | `net.egress.cdn-rstudio`, `net.egress.cdn-posit` | blocking |
| Outbound networking — license activation | `net.egress.license-activation` | warning |
| R installed under `/opt/R/<version>` | `workbench.r.versioned-install` | warning |
| Python installed under `/opt/python/<version>` | `workbench.python.versioned-install` | warning |
| System dependencies (build deps for R/Python packages) | `pkg.openssl-dev`, `pkg.libcurl-dev`, `pkg.libxml2-dev`, `pkg.gdebi.ubuntu` | warning |

## Package Manager

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for PPM hostname (customer-supplied) | `ppm.ssl.cert-key-match` | blocking |
| Outbound to Posit Package Service | `ppm.egress.sync` | blocking |
| Sufficient disk for package cache | `sizing.packagemanager.recommended` | warning |

## Connect

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for Connect hostname (customer-supplied) | `connect.ssl.cert-key-match` | blocking |
| Outbound email server access | `connect.smtp.reachable` | warning |
| Quarto installed at `/opt/quarto/<version>` | `connect.quarto.present` | warning |

## Workbench

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for Workbench hostname (customer-supplied) | `workbench.ssl.cert-key-match` | blocking |
| Quarto available on PATH | `workbench.quarto.present` | warning |
| IdP metadata or discovery URL reachable | `workbench.idp.metadata` | warning |
| Desired editors identified | _(human-only checklist)_ | — |

## Out of scope (handled by `vip`, not pev)

These were intentionally removed from the catalog because they validate state that only exists *after* a Posit product has been installed, which is `vip`'s job:

- `*.license.activated` — `rstudio-server license-manager status` and the Connect/PPM equivalents
- `*.license.file-present` — globbing `/var/lib/rstudio-*/*.lic`
- `*.binary.present` — `/usr/lib/rstudio-server/bin/rserver`, `/opt/rstudio-connect/bin/rstudio-connect`, `/opt/rstudio-pm/bin/rstudio-pm`
- `workbench.ssl.config` — parsing `/etc/rstudio/rserver.conf` for `ssl-enabled=1`
- Anything that depends on `rstudio-server`, `rstudio-connect`, or `rstudio-pm` being installed

## When to add a row

Whenever you add a YAML check whose origin is a runbook prereq, add a row here in the same PR. A check is in scope for pev only if a customer can satisfy it *before* installing any Posit product. If satisfaction requires the product to already be installed, it belongs in `vip`.
