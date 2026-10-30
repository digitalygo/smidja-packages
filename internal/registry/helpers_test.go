package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const emptyIndexJSON = "{\n  \"schemaVersion\": 2,\n  \"packages\": []\n}\n"

const emptyCatalogMD = `# Smidja package catalog

Generated discovery index of [Smidja](https://github.com/digitalygo/smidja) runtime packages. Listing is not endorsement. Packages may contain executable tools; inspect and trust the publisher before installing. This registry neither hosts packages nor verifies their runtime contents.

Choose a release feed for your exact OS/architecture and configure that HTTPS source in the manager or client. The current runtime does not automatically select targets or install from this catalog. Publisher feeds, not this index, supply versions and archive checksums. Private packages stay outside this public catalog.

Submission rules: [contribution guide](.github/CONTRIBUTING.md). Schema: [registry format](docs/registry-format.md).

## Packages

No packages are listed yet. The registry starts empty on purpose; see the [contribution guide](.github/CONTRIBUTING.md) to submit the first descriptor.
`

const validDescriptorBody = `"schemaVersion":2,"id":"example.package","scopes":["manager"],"releaseFeeds":{"linux/amd64":"https://example.org/release.json"},"repository":"a/b","description":"A test package.","maintainers":["alice"]`

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newEmptyRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, PackagesDir, KeepFile), "")
	return root
}

func newPopulatedRepo(t *testing.T) string {
	t.Helper()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatalf("remove keep file: %v", err)
	}
	writeDescriptor(t, root, "acme/tools")
	writeDescriptor(t, root, "Octo-Labs/Daily-Notes")
	return root
}

func writeDescriptor(t *testing.T, root, repository string) string {
	t.Helper()
	relative := filepath.Join(PackagesDir, filepath.FromSlash(strings.ToLower(repository))+".json")
	writeFile(t, filepath.Join(root, relative), descriptorFor(t, repository))
	return relative
}

func descriptorFor(t *testing.T, repository string) string {
	t.Helper()
	data := marshalDescriptor(t, map[string]any{
		"schemaVersion": 2,
		"repository":    repository,
		"description":   "A test package.",
		"maintainers":   []string{"alice"},
	})
	return string(data)
}

func assertNoTempFiles(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{IndexFile, CatalogFile} {
		matches, err := filepath.Glob(filepath.Join(root, "."+name+".tmp-*"))
		if err != nil {
			t.Fatalf("glob temp files: %v", err)
		}
		if len(matches) > 0 {
			t.Fatalf("temp files left behind: %v", matches)
		}
	}
}
