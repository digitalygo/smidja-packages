# Agents quality baseline

When edits touch the codebase, run the tests that cover the changed files before you finish. Fix every failure and re-run until all delta tests pass.

## Delta tests

- Map every behavior change to at least one test and run those tests, plus anything upstream they exercise.
- All mapped delta tests must pass. Never weaken, skip, or delete a test to make a run green.
- Failures outside the delta do not block. Report them as out-of-scope signals instead of fixing them inside an unrelated change.
- Keep coverage between 80% and 100% on the executable lines the delta adds or modifies, and report the coverage result with the change. For non-executable changes such as configuration, documentation, or generated artifacts, mark tests and coverage N/A with a short justification.

## File length

- Treat length as a contextual maintainability control, not a numerical cap. Judge it together with cohesion and separability.
- Production code normally falls in the 500 to 700 line range. Cohesive tests may reasonably approach 1000 lines.
- Never thin tests, drop cases, or split cohesive fixtures or long inputs merely to reduce length.
- Split a file across multiple files only when the file shows materially poor navigation or maintainability and a meaningful split by behavior or responsibility exists.

## Code comments

- Never write code comments. Clear naming and structure explain the code.
- If code seems to need a comment, simplify it or split it into smaller files instead.
- Use descriptive names for variables, functions, and files in place of annotation.

## Registry invariants

- This is public discovery for runtime packages, not a host, updater, crawler or executor. Never add private package metadata.
- Descriptor schema 2 is independent of runtime manifest schema 1. IDs, scopes and explicit per-target public HTTPS release feeds are discovery metadata. Versions, hashes, archives and payloads remain publisher-owned and are not mirrored.
- The runtime takes one explicit source URL. Do not claim automatic target selection, catalog-dependent installation or click-to-install integration.
- Supported targets are Linux/macOS amd64 and arm64 only.
- `index.json` and `catalog.md` are generated files. Change them only with `go run ./cmd/registry generate`; `go run ./cmd/registry check` must pass.
- Empty catalog is valid: only `packages/.gitkeep`, no invented listings.
- CI keeps `contents: read`, `persist-credentials: false`, full action SHAs, no secrets, and no `pull_request_target`.
