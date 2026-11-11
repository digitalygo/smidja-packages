# Runtime package discovery format

Descriptor schema **2** is independent of the runtime archive manifest schema **1**. The previous content-only descriptor schema 1 is superseded; the public catalog was empty, so no listings need migration. This repository never reads or validates remote manifests and never mirrors versions or checksums.

## Descriptor

Place one JSON file at `packages/<lowercase-owner>/<lowercase-repo>.json`, matching `repository` case-insensitively. Preserve the publisher's repository casing in the JSON. Duplicate repository identities, symlinks, special files and unexpected directory structure are rejected. Keep only `packages/.gitkeep` when there are no listings.

Illustrative example only; this is not an actual listing:

```json
{
  "schemaVersion": 2,
  "repository": "example/coding-tools",
  "description": "Review Verk and supporting tools.",
  "maintainers": ["alice"],
  "topics": ["review"],
  "id": "example.coding",
  "scopes": ["manager", "client"],
  "releaseFeeds": {
    "linux/amd64": "https://publisher.example/linux-amd64/release.json",
    "linux/arm64": "https://publisher.example/linux-arm64/release.json",
    "darwin/amd64": "https://publisher.example/darwin-amd64/release.json",
    "darwin/arm64": "https://publisher.example/darwin-arm64/release.json"
  }
}
```

Required fields:

- `schemaVersion`: integer 2.
- `repository`: GitHub `owner/repo`; owner length 1–39, alphanumeric segments separated by single hyphens; repository length 1–100, ASCII letters, digits, underscores, hyphens or dots, excluding `.` and `..`.
- `description`: trimmed single line, 1–200 Unicode characters; no control or format characters.
- `maintainers`: 1–10 distinct lowercase GitHub login identifiers, at most 39 characters each.
- `id`: runtime package identity, matching `[a-z0-9][a-z0-9.-]{0,127}`. It must match the feed and archive manifest.
- `scopes`: `manager`, `client`, or both, without duplicates. The archive manifest must support the advertised scopes.
- `releaseFeeds`: one to four entries keyed by `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`. Values are HTTPS source URLs without embedded credentials, query strings or fragments. Each source must serve a compatible archive for that exact target. A portable package may reuse a feed across targets if its manifest supports them all.

Optional `topics`: zero to five distinct lowercase topic identifiers, at most 35 characters each, matching `[a-z0-9]+(-[a-z0-9]+)*`.

Feeds must be public and unauthenticated. Do not add private metadata or credentials. The syntax validator does not make network requests or prove endpoint availability, ownership, compatibility or safety. Admission review checks publisher ownership and advertised ID/scope/target claims.

Unknown or duplicate keys (including escaped aliases), nulls, wrong types, trailing data, malformed UTF-8 and descriptors over 16 KiB are rejected. A run accepts at most 10,000 descriptors. Invalid input leaves generated outputs untouched.

## Runtime installation contract

The user chooses one target-specific feed and configures that HTTPS source in Smidja manager/client. There is no automatic target selector or catalog install integration. The legacy `smidja pkg install owner/repo@version` command does not install these archives.

The feed returns `{ "id", "version", "archive_url", "sha256" }`; the ZIP contains `smidja.json` and its SHA256-inventoried payload. Runtime versions are stable `x.y.z`; minimum runtime constraints use `>=x.y.z`. Installed packages are immutable. Publisher updates are checked by the runtime, not by this registry. No version bump in discovery is needed when a publisher updates its feed.

Listing does not authorize executing package contents. Packages may contain native executable helpers. Inspect and trust the publisher before installation. Credentials are configured locally by the user; absent API credentials produce invocation errors, not account provisioning.

## Generated outputs

Run `go run ./cmd/registry generate`, then `go run ./cmd/registry check`.

`index.json` has top-level schemaVersion 2 and a packages array. Each record contains repository, description, maintainers, topics, id, scopes and releaseFeeds. Records sort case-insensitively by repository. Array metadata and feed keys are sorted; JSON is two-space indented with one trailing newline, no timestamps or environment data.

`catalog.md` includes repository/tag links, escaped descriptions, identity, scopes and escaped feed URLs. Feed URLs are displayed as text, not promoted to trusted executable actions. All ASCII punctuation in user-provided rendered text is escaped. Generated output is deterministic and reviewed with descriptors.

`check` renders in memory, compares exact bytes and never writes. CI remains offline, read-only and secret-free; it never executes package payloads.
