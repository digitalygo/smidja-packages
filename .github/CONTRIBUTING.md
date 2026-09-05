# Contributing

Thank you for wanting to list a package. This registry is a small, low maintenance index: a pull request adds one descriptor, the generated files are regenerated, and the registry maintainers review the submission.

## What can be listed

- Content-only smidja packages: public GitHub repositories of skills, agents, prompts, and config defaults with no code.
- Compiled bundles are not listed here. The registry does not host packages, mirror versions, or certify safety, and listing is discovery, not endorsement.

## Submission rules

### Initial listing

- A pull request that lists a package must come from someone with authority over that package.
- Personal submissions: the pull request author must own the package repository.
- Organization or delegated submissions: the pull request must link public upstream authorization, for example an issue or discussion in the package repository where a maintainer of record approves the listing.
- Evidence stays in the pull request discussion or in the package's own public release metadata. Never add evidence to the descriptor or to any package archive.

### Updates

- An update to an existing descriptor requires the pull request author to be a maintainer already listed in the base descriptor.

### Transfers and replacements

- Moving a listing to a new repository, or replacing its maintainers, requires renewed ownership evidence under the initial listing rules.

## Descriptor

- Add `packages/<lowercase-owner>/<lowercase-repo>.json` following the schema documented in the [registry format](../docs/registry-format.md).
- Regenerate the output files with `go run ./cmd/registry generate`.
- Validate the regenerated outputs with `go run ./cmd/registry check`.
- Commit the descriptor and the generated output files in the same pull request. CI fails when the committed outputs do not match the descriptors.

## Review and admission

- Registry maintainers review the descriptor and the ownership evidence.
- Admission includes verifying one explicit canonical version tag of the package in a disposable credential-free environment before merging.
- The repository CI runs offline checks only: it never crawls the network and never executes package content.

## Pull requests

- Keep CI on `contents: read`, pinned action SHAs, and no secrets. Workflow changes require a registry maintainer review.
- When the last descriptor is removed, restore `packages/.gitkeep` so the empty state stays valid.

## License

MIT. By contributing you agree that your contributions are licensed under the repository license.
