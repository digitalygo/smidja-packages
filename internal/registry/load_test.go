package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestLoadEmptyRegistry(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	descriptors, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(descriptors) != 0 {
		t.Fatalf("Load returned %d descriptors, want 0", len(descriptors))
	}
}

func TestLoadMissingPackagesDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), filepath.Join(root, PackagesDir)) {
		t.Fatalf("Load error = %v, want a missing packages directory error naming the path", err)
	}
}

func TestLoadPackagesRootSymlink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, PackagesDir)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Load error = %v, want a symlink rejection", err)
	}
}

func TestLoadOwnerDirectorySymlink(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	real := filepath.Join(root, PackagesDir, "real-owner")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, PackagesDir, "owner")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Load error = %v, want a symlink rejection", err)
	}
}

func TestLoadDescriptorFileSymlink(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, PackagesDir, "owner"), 0o755); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(t.TempDir(), "real.json")
	if err := os.WriteFile(real, []byte(descriptorFor(t, "owner/bar")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(root, PackagesDir, "owner", "bar.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Load error = %v, want a symlink rejection", err)
	}
}

func TestLoadBrokenSymlink(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing.json"), filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("Load error = %v, want a symlink rejection", err)
	}
}

func TestLoadPackagesRootIsFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, PackagesDir), "not a directory")
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("Load error = %v, want a not a directory rejection", err)
	}
}

func TestLoadUnreadableOwnerDirectory(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	ownerDir := filepath.Join(root, PackagesDir, "owner")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ownerDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(ownerDir, 0o755); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "walk") {
		t.Fatalf("Load error = %v, want a walk error", err)
	}
}

func TestLoadUnreadableDescriptor(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, PackagesDir, "a", "b.json")
	writeFile(t, path, descriptorFor(t, "a/b"))
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "read "+path) {
		t.Fatalf("Load error = %v, want a read error naming the path", err)
	}
}

func TestLoadSpecialFile(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := syscall.Mkfifo(filepath.Join(root, PackagesDir, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "special files") {
		t.Fatalf("Load error = %v, want a special file rejection", err)
	}
}

func TestLoadUnexpectedFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		paths   []string
		wantErr string
	}{
		{"root level readme", []string{"README.md"}, "is unexpected"},
		{"root hidden file", []string{".hidden"}, "is unexpected"},
		{"owner level text file", []string{"owner/notes.txt"}, "descriptors are .json files"},
		{"owner level markdown", []string{"owner/bar.md"}, "descriptors are .json files"},
		{"empty stem", []string{"owner/.json"}, "empty descriptor name"},
		{"uppercase extension", []string{"owner/bar.JSON"}, "descriptors are .json files"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := newEmptyRepo(t)
			for _, path := range test.paths {
				writeFile(t, filepath.Join(root, PackagesDir, filepath.FromSlash(path)), "x")
			}
			if _, err := Load(root); err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Load error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestLoadWrongDepth(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, PackagesDir, "owner", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, PackagesDir, "owner", "sub", "deep.json"), descriptorFor(t, "owner/deep"))
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unexpected directory") {
		t.Fatalf("Load error = %v, want an unexpected directory rejection", err)
	}
}

func TestLoadDescriptorDirectory(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, PackagesDir, "owner", "bar.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "unexpected directory") {
		t.Fatalf("Load error = %v, want an unexpected directory rejection", err)
	}
}

func TestLoadEmptyOwnerDirectoryAllowed(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.MkdirAll(filepath.Join(root, PackagesDir, "owner"), 0o755); err != nil {
		t.Fatal(err)
	}
	descriptors, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(descriptors) != 0 {
		t.Fatalf("Load returned %d descriptors, want 0", len(descriptors))
	}
}

func TestLoadKeepFileRules(t *testing.T) {
	t.Parallel()
	t.Run("missing while empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, PackagesDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "is required while no descriptors are listed") {
			t.Fatalf("Load error = %v, want a missing keep file error", err)
		}
	})
	t.Run("present with descriptors", func(t *testing.T) {
		t.Parallel()
		root := newEmptyRepo(t)
		writeDescriptor(t, root, "a/b")
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "must be removed once descriptors are listed") {
			t.Fatalf("Load error = %v, want a keep file rejection", err)
		}
	})
}

func TestLoadPathIdentity(t *testing.T) {
	t.Parallel()
	t.Run("lowercase path with mixed-case repository", func(t *testing.T) {
		t.Parallel()
		root := newEmptyRepo(t)
		if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, PackagesDir, "octo-labs", "daily-notes.json"), descriptorFor(t, "Octo-Labs/Daily-Notes"))
		descriptors, err := Load(root)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if len(descriptors) != 1 {
			t.Fatalf("Load returned %d descriptors, want 1", len(descriptors))
		}
		if descriptors[0].Repository != "Octo-Labs/Daily-Notes" {
			t.Fatalf("descriptor repository = %q, want the mixed case preserved", descriptors[0].Repository)
		}
	})
	t.Run("uppercase path rejected", func(t *testing.T) {
		t.Parallel()
		root := newEmptyRepo(t)
		if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, PackagesDir, "O", "R.json"), descriptorFor(t, "o/r"))
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "does not match the lowercase repository") {
			t.Fatalf("Load error = %v, want a lowercase path error", err)
		}
	})
	t.Run("path does not match repository", func(t *testing.T) {
		t.Parallel()
		root := newEmptyRepo(t)
		if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(root, PackagesDir, "foo", "baz.json"), descriptorFor(t, "foo/bar"))
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "does not match the lowercase repository") {
			t.Fatalf("Load error = %v, want a path identity error", err)
		}
	})
}

func TestCheckCollisions(t *testing.T) {
	t.Parallel()
	t.Run("case insensitive collision", func(t *testing.T) {
		t.Parallel()
		err := checkCollisions([]Descriptor{
			{SchemaVersion: 2, Repository: "acme/tools", Description: "d", Maintainers: []string{"a"}},
			{SchemaVersion: 2, Repository: "ACME/TOOLS", Description: "d", Maintainers: []string{"a"}},
		})
		if err == nil || !strings.Contains(err.Error(), "collide case insensitively") {
			t.Fatalf("checkCollisions error = %v, want a collision error", err)
		}
		if !strings.Contains(err.Error(), `"acme/tools"`) || !strings.Contains(err.Error(), `"ACME/TOOLS"`) {
			t.Fatalf("collision error %v does not name both repositories", err)
		}
	})
	t.Run("distinct repositories pass", func(t *testing.T) {
		t.Parallel()
		if err := checkCollisions([]Descriptor{
			{SchemaVersion: 2, Repository: "acme/tools", Description: "d", Maintainers: []string{"a"}},
			{SchemaVersion: 2, Repository: "octo-labs/notes", Description: "d", Maintainers: []string{"a"}},
		}); err != nil {
			t.Fatalf("checkCollisions rejected distinct repositories: %v", err)
		}
	})
}

func TestLoadDescriptorTooLarge(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Remove(filepath.Join(root, PackagesDir, KeepFile)); err != nil {
		t.Fatal(err)
	}
	big := `{"schemaVersion":2,"id":"example.package","scopes":["manager"],"releaseFeeds":{"linux/amd64":"https://example.org/release.json"},"repository":"a/b","description":"d","maintainers":["u"]}` + strings.Repeat(" ", MaxDescriptorBytes+1)
	writeFile(t, filepath.Join(root, PackagesDir, "a", "b.json"), big)
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "the limit is 16384 bytes") {
		t.Fatalf("Load error = %v, want a size limit error", err)
	}
}

func TestLoadMaxDescriptors(t *testing.T) {
	t.Parallel()
	t.Run("at the limit", func(t *testing.T) {
		t.Parallel()
		root := writeBulkRegistry(t, MaxDescriptors)
		descriptors, err := Load(root)
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if len(descriptors) != MaxDescriptors {
			t.Fatalf("Load returned %d descriptors, want %d", len(descriptors), MaxDescriptors)
		}
	})
	t.Run("over the limit", func(t *testing.T) {
		t.Parallel()
		root := writeBulkRegistry(t, MaxDescriptors+1)
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("more than %d descriptors", MaxDescriptors)) {
			t.Fatalf("Load error = %v, want a descriptor count error", err)
		}
	})
}

func writeBulkRegistry(t *testing.T, count int) string {
	t.Helper()
	root := t.TempDir()
	ownerDir := filepath.Join(root, PackagesDir, "bulk")
	if err := os.MkdirAll(ownerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < count; index++ {
		name := fmt.Sprintf("r%d.json", index)
		content := fmt.Sprintf(`{"schemaVersion":2,"id":"example.package","scopes":["manager"],"releaseFeeds":{"linux/amd64":"https://example.org/release.json"},"repository":"bulk/r%d","description":"d","maintainers":["a"]}`, index)
		if err := os.WriteFile(filepath.Join(ownerDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestFindRoot(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	writeFile(t, filepath.Join(root, "go.mod"), "module github.com/digitalygo/smidja-packages\n\ngo 1.26.0\n")
	nested := filepath.Join(root, PackagesDir, "owner")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := FindRoot(nested)
	if err != nil {
		t.Fatalf("FindRoot returned error: %v", err)
	}
	if found != root {
		t.Fatalf("FindRoot = %s, want %s", found, root)
	}
	if _, err := FindRoot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "could not find the repository root") {
		t.Fatalf("FindRoot error = %v, want a root not found error", err)
	}
}
