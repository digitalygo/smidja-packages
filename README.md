# Smidja packages

A public discovery registry of content-only [smidja](https://github.com/digitalygo/smidja) packages. The registry lists where packages live so that people can find them. It is not package hosting, it is not a compiled bundle registry, and it is not part of the install path.

Content-only packages are versioned public GitHub repositories of skills, agents, prompts, and config defaults that `smidja pkg` installs into an existing harness. Installation stays direct and registry independent:

```bash
smidja pkg install owner/repo@version
```

GitHub tags remain the authoritative record of versions. Cutting a release never requires a registry update, and the registry never mirrors versions, manifests, checksums, or package content.

## What is inside

- `packages/<lowercase-owner>/<lowercase-repo>.json`, one publisher maintained descriptor per listed package.
- `index.json` and `catalog.md`, generated files committed to git.
- `cmd/registry`, a standard library only Go tool that validates the descriptors and produces the generated files.
- The [registry format](docs/registry-format.md) documents the descriptor schema, the strict parsing rules, and the generated output contracts.

## Listing is discovery, not endorsement

The registry does not certify the authenticity, safety, or quality of any listed package, and content-only prompts and config can influence agent behavior. Review a package and its tags before installing it. Admission is handled by the registry maintainers, who verify one explicit canonical tag in a disposable credential-free environment. The required CI never crawls the network and never executes package content.

## Commands

From a checkout of this repository:

```bash
go run ./cmd/registry generate
go run ./cmd/registry check
```

`generate` validates every descriptor and rewrites `index.json` and `catalog.md`. `check` validates the descriptors and verifies that the committed outputs match exactly without writing anything. CI runs `check` on every pull request and every push to main.

## Submitting a package

Open a pull request that adds one descriptor and the regenerated output files. The [contribution guide](.github/CONTRIBUTING.md) covers the ownership evidence rules for initial listings, updates, transfers, and replacements.

## Repository layout

```text
packages/<owner>/<repo>.json   publisher maintained descriptors
packages/.gitkeep              placeholder while no descriptors are listed
index.json                     generated machine readable index
catalog.md                     generated human readable catalog
cmd/registry                   the registry command line tool
internal/registry              validation, rendering, and output logic
docs/registry-format.md        the descriptor schema and output contracts
```
