# Package registry submission

## Summary

A short description of the change: which package is listed, updated, transferred, or removed.

## Descriptor

- [ ] The descriptor sits at `packages/<lowercase-owner>/<lowercase-repo>.json`
- [ ] `go run ./cmd/registry check` passes locally
- [ ] The output of `go run ./cmd/registry generate` is committed

## Evidence

- [ ] Initial listing: I own the package repository, or this pull request links public upstream authorization from a maintainer of record
- [ ] Update: I am a maintainer listed in the base descriptor
- [ ] Transfer or replacement: renewed ownership evidence is linked in this pull request

## Scope

- [ ] The package is content only
- [ ] The change adds no package content, versions, hashes, manifests, or licenses to the registry
- [ ] Evidence lives in this pull request discussion or in public release metadata, never in the descriptor
