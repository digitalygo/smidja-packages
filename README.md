# Smidja packages

Public discovery for [Smidja](https://github.com/digitalygo/smidja) manager/client runtime packages: teams, workflows, skills and extension tools, including executable helpers. Smidja itself remains the base binary; packages are not replacement harness binaries.

This repository neither hosts packages nor downloads or executes them. It does not mirror versions, archives, manifests or checksums. The empty catalog is intentional; no package is listed until its publisher submits a reviewed descriptor. Private package metadata must never be submitted here.

## Discovery and installation

Descriptors provide a package ID and explicit HTTPS release feed for each supported OS/architecture. Choose the feed matching the machine where the package will execute, then configure that source in the manager or client. Linux/macOS amd64 and arm64 are supported; ARMv7 and Windows are not supported initially.

The current runtime accepts one explicit source URL; it does not automatically select a target from this catalog, and there is no click-to-install integration. Do not use the legacy `smidja pkg install owner/repo@version` command for these runtime archives.

A publisher feed returns `id`, `version`, `archive_url`, and `sha256`. Its ZIP contains `smidja.json` runtime manifest schema 1 plus the inventoried payload. The runtime verifies package compatibility and hashes; this offline discovery validator does not. Public feeds must work without credentials. Private sources use the same package format but are configured directly, outside this public catalog.

Installing trusts the publisher and its automatic updates. Review source and package contents before trusting them: tools may execute code. Missing tool credentials are reported on invocation, not provisioned by this catalog.

## Maintenance

```bash
go test ./...
go run ./cmd/registry generate
go run ./cmd/registry check
```

Descriptors live in `packages/<lowercase-owner>/<lowercase-repo>.json`. `index.json` and `catalog.md` are generated, deterministic outputs. CI is offline, read-only and secret-free. See [format](docs/registry-format.md) and [contributing](.github/CONTRIBUTING.md).
