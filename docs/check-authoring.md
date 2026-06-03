# Authoring custom YAML check packs

`pev` accepts custom YAML packs via `--checks-file path.yaml` (repeatable) and from `~/.config/pev/checks.d/*.yaml` (when `--include-user-checks` is set, default true). Packs use the same schema as the built-in catalog under `/checks`.

## Minimum viable pack

```yaml
schema_version: 1
checks:
  - id: mycorp.example.binary
    title: My corp's wrapper binary is installed
    severity: warning
    primitive: file
    why: |
      Our prod images ship a wrapper at /opt/mycorp/bin/wrap. If it's missing,
      Posit content can't authenticate against our internal vault.
    with:
      path: /opt/mycorp/bin/wrap
      must_exist: true
```

Validate it before shipping:

```bash
pev lint-checks ./mycorp-pack.yaml
```

## All check fields

| Field | Required | Description |
|------|----------|-------------|
| `id` | yes | Globally unique; convention `<area>.<topic>.<facet>`. |
| `title` | yes | One-line, present-tense; appears in tables and the Markdown report. |
| `severity` | yes | `blocking` \| `warning` \| `info`. |
| `tags` | no | Free-form labels for `--tags`/`--skip-tags`. |
| `applies_to.os` | no | Canonical OS ids: `ubuntu-22.04`, `ubuntu-24.04`, `rhel-8`, `rhel-9`, `rhel-10`. |
| `applies_to.products` | no | `workbench` \| `connect` \| `packagemanager`. |
| `applies_to.arch` | no | `amd64` \| `arm64`. |
| `applies_to.roles` | no | Reserved for future use. |
| `requires_root` | no | If true and pev runs as non-root, the check is `SKIPPED`. |
| `why` | yes | Rationale shown to users; two sentences, plain English. |
| `remediation` | no | Free-text fix hint; reserved for the v2 `pev fix` flow. |
| `references` | no | List of authoritative doc URLs. |
| `primitive` | yes | One of the registered primitive names — see `docs/primitives.md`. |
| `with` | yes | Primitive-specific payload. |

## Templating

Any string in `with:` may use Go `text/template` syntax against:

- `{{ .Facts.Hostname }}`, `{{ .Facts.FQDN }}`, `{{ .Facts.OS }}`, `{{ .Facts.Arch }}`, `{{ .Facts.CPUs }}`, `{{ .Facts.MemMB }}`
- `{{ .Inputs.<key> }}` — populated by CLI flags (`--license-file`, `--hostnames`, `--idp`) and (in v0.2+) survey prompts

Missing-key errors during template expansion become `SKIPPED (missing or invalid input: <details>)` rather than aborting the run.

## ID conventions

- Use lower-case, dotted segments: `<area>.<topic>.<facet>`.
- Areas: `os`, `net`, `sizing`, `pkg`, `workbench`, `connect`, `ppm`, `mycorp`.
- Avoid timestamps and version numbers in IDs — checks evolve, IDs persist.

## Severity guidance

- **blocking** — install will fail or be unusable until fixed.
- **warning** — install will succeed but something will break later (perf, missing deps for compiling R/Python packages, etc.).
- **info** — useful diagnostic; never gates anything.

## Tagging guidance

Tags shape `--tags`/`--skip-tags` filters. Reuse what's in the built-in catalog where applicable: `os`, `network`, `egress`, `license`, `ssl`, `sizing`, `packages`, `auth`, `quarto`, `r`, `python`. Add corp-specific prefixes like `mycorp.*` for tags you don't expect to share upstream.
