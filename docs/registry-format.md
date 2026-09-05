# Registry format

The registry lists content-only [smidja](https://github.com/digitalygo/smidja) packages. Each listed package is described by one publisher maintained JSON descriptor under `packages/`, and the generated `index.json` and `catalog.md` files are derived from those descriptors. The descriptor schema version 1 is independent from the package manifest schema 0 that `smidja.json` uses inside packages; the registry never reads, mirrors, or validates package manifests.

## What the registry is not

- The registry does not host package content, mirror versions, tags, manifests, or checksums, or crawl GitHub.
- `smidja pkg install owner/repo@version` is direct and registry independent, and GitHub tags remain the authoritative record of versions.
- Listing is discovery, not endorsement: the registry does not certify authenticity, safety, or quality, and content-only prompts and config can influence agent behavior.

## Descriptor location

`packages/<lowercase-owner>/<lowercase-repo>.json`

- The descriptor path must be exactly `<owner>/<repo>.json` under `packages/` with both segments lowercase, matching the `repository` field after lowercasing. Uppercase segments are rejected, for example `packages/Octo-Labs/Tools.json` fails and only `packages/octo-labs/tools.json` is accepted.
- The `repository` field keeps its original casing in the generated output because harness identity comparison is case sensitive, so a descriptor at `packages/octo-labs/tools.json` may declare `Octo-Labs/Tools`.
- Two repositories that are equal case insensitively collide and are rejected, for example `acme/tools` and `ACME/TOOLS`; both resolve to the same canonical descriptor path, so a collision can never be reached through distinct files.
- While no descriptors are listed, the directory holds only `packages/.gitkeep`; once a descriptor lands, `.gitkeep` must be removed.
- Symlinks, special files, unexpected files, and any structure other than `packages/<owner>/<repo>.json` are rejected.

## Descriptor schema

Example:

```json
{
  "schemaVersion": 1,
  "repository": "octo-labs/smidja-daily-notes",
  "description": "Daily notes skill with its config defaults.",
  "maintainers": ["alice"],
  "topics": ["notes", "productivity"]
}
```

| Field | Type | Required | Constraints |
| --- | --- | --- | --- |
| `schemaVersion` | number | yes | Must be the number `1`. |
| `repository` | string | yes | `owner/repo`. The owner is alphanumeric segments separated by single hyphens with 1 to 39 characters. The repository name is 1 to 100 ASCII letters, digits, underscores, hyphens, and dots, and `.` or `..` are rejected. Schemes, hosts, versions, queries, and fragments cannot survive these rules. |
| `description` | string | yes | Trimmed, single line, plain text with 1 to 200 Unicode characters and no control, line separator, or format characters. |
| `maintainers` | array | yes | 1 to 10 distinct lowercase GitHub logins of at most 39 characters: lowercase letters and digits in segments separated by single hyphens. |
| `topics` | array | no | Absent, empty, or up to 5 distinct lowercase topic identifiers of at most 35 characters, matching `[a-z0-9]+(-[a-z0-9]+)*`. |

Descriptors deliberately carry no global package id, versions, latest pointer, minimum harness, dependencies, hashes, manifests, licenses, or URLs. Package identity is the repository, and package versions are the GitHub tags.

## Strict parsing

The parser rejects:

- Unknown keys and key case aliases, so `SchemaVersion` is as unknown as a misspelled field.
- Duplicate keys, including keys that only become duplicates after JSON escape decoding, so a `\u0073chemaVersion` key colliding with `schemaVersion` is rejected.
- JSON `null` for any field or array entry.
- Wrong value types.
- Trailing JSON data after the descriptor object.
- Invalid UTF-8, control characters, and Unicode format characters such as bidi overrides, isolates, the word joiner, and the zero-width space.
- Descriptors larger than 16384 bytes.
- More than 10000 descriptors in one run.

An invalid descriptor leaves `index.json` and `catalog.md` untouched.

## Generated outputs

Both files are produced only by the registry tool and are committed to git.

### index.json

- A top-level object with `schemaVersion` set to `1` and an always present `packages` array.
- One record per descriptor with exactly the fields `repository`, `description`, `maintainers`, and `topics`, in that order.
- Records sort case insensitively by `repository`; `maintainers` and `topics` sort ascending inside each record.
- Two-space indented JSON with one trailing newline, no timestamps, and no environment data. HTML significant characters in strings are escaped as JSON escapes.

### catalog.md

- One H1, a non-endorsement warning, a link to the contribution guide, and an explicit empty state while no packages are listed.
- One entry per descriptor with a repository link and a tags link under `https://github.com/<owner>/<repo>`, sorted like the index records.
- Descriptions are untrusted text: every ASCII punctuation character in a description is backslash escaped before it is rendered, so descriptions cannot inject markdown, HTML, or links. Format characters never reach rendering because the parser rejects them first. Generated links derive only from validated identifiers.

## Commands

From a checkout of this repository:

```bash
go run ./cmd/registry generate
go run ./cmd/registry check
```

- `generate` validates every descriptor first and then rewrites `index.json` and `catalog.md` through same-directory temporary files and renames. Errors name the affected path.
- `check` validates the descriptors, renders both outputs in memory, and compares the exact committed bytes. It reports missing or stale output and never writes. CI runs `check` on every pull request and every push to main.
