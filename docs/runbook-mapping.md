# Runbook prereq → check ID mapping

Each row in the customer onboarding runbook (the BCBS-style "Prerequisites" tab) maps to one or more pev built-in checks. This file is the contract — when a runbook prereq lands here, the corresponding YAML check exists in the catalog and is exercised by CI.

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
| Posit license obtained, path documented | `workbench.license.file-present`, `connect.license.file-present`, `ppm.license.file-present` | info |
| System dependencies (build deps for R/Python packages) | `pkg.openssl-dev`, `pkg.libcurl-dev`, `pkg.libxml2-dev`, `pkg.gdebi.ubuntu` | warning |

## Package Manager

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for PPM hostname | `ppm.ssl.cert-key-match` | blocking |
| Outbound to Posit Package Service | `ppm.egress.sync` | blocking |
| Sufficient disk for package cache | `sizing.packagemanager.recommended` | warning |
| License activated | `ppm.license.activated`, `ppm.license.file-present` | blocking, info |

## Connect

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for Connect hostname | `connect.ssl.cert-key-match` | blocking |
| Outbound email server access | `connect.smtp.reachable` | warning |
| Quarto installed | `connect.quarto.present` | warning |
| License activated | `connect.license.activated`, `connect.license.file-present` | blocking, info |
| Authentication provider details gathered | _(human-only checklist)_ | — |

## Workbench

| Runbook prereq | pev check ID(s) | Severity |
|---|---|---|
| SSL certificate for Workbench hostname | `workbench.ssl.cert-key-match`, `workbench.ssl.config` | blocking, warning |
| Quarto installed | `workbench.quarto.present` | warning |
| Authentication provider details gathered | `workbench.idp.metadata` | warning |
| Desired editors identified | _(human-only checklist)_ | — |
| License activated | `workbench.license.activated`, `workbench.license.file-present` | blocking, info |

## When to add a row

Whenever you add a YAML check whose origin is a runbook prereq, add a row here in the same PR. CI runs `make lint` against the catalog; this file is checked manually during PR review.
