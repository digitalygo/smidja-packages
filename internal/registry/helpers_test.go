package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const emptyIndexJSON = "{\n  \"schemaVersion\": 1,\n  \"packages\": []\n}\n"

const emptyCatalogMD = "# Smidja package catalog\n" +
	"\n" +
	"Generated discovery index of content-only [smidja](https://github.com/digitalygo/smidja) packages. This listing is not an endorsement: the registry does not host package content, does not mirror versions, and does not certify the authenticity, safety, or quality of any listed package. Content-only packages carry prompts, agents, and config defaults that can influence agent behavior, so review a package and its tags before installing it.\n" +
	"\n" +
	"Installation is direct and registry independent, and GitHub tags remain the authoritative record of versions:\n" +
	"\n" +
	"```text\n" +
	"smidja pkg install owner/repo@version\n" +
	"```\n" +
	"\n" +
	"To list a package, open a pull request that adds a descriptor under `packages/` with the required ownership evidence. The submission rules live in the [contribution guide](.github/CONTRIBUTING.md), and the descriptor schema is documented in the [registry format](docs/registry-format.md).\n" +
	"\n" +
	"## Packages\n" +
	"\n" +
	"No packages are listed yet. The registry starts empty on purpose; see the [contribution guide](.github/CONTRIBUTING.md) to submit the first descriptor.\n"

const validDescriptorBody = `"schemaVersion":1,"repository":"a/b","description":"A test package.","maintainers":["alice"]`

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
		"schemaVersion": 1,
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
