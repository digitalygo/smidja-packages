package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readOutput(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func TestGenerateEmpty(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	count, err := Generate(root)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("Generate count = %d, want 0", count)
	}
	if got := readOutput(t, root, IndexFile); got != emptyIndexJSON {
		t.Fatalf("index.json = %q, want %q", got, emptyIndexJSON)
	}
	if got := readOutput(t, root, CatalogFile); got != emptyCatalogMD {
		t.Fatalf("catalog.md = %q, want %q", got, emptyCatalogMD)
	}
	assertNoTempFiles(t, root)
}

func TestGeneratePopulated(t *testing.T) {
	t.Parallel()
	root := newPopulatedRepo(t)
	count, err := Generate(root)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("Generate count = %d, want 2", count)
	}
	index := BuildIndex(mustLoad(t, root))
	catalog, err := BuildCatalog(mustLoad(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if got := readOutput(t, root, IndexFile); got != string(index) {
		t.Fatalf("index.json does not match the rendered index:\n%s", got)
	}
	if got := readOutput(t, root, CatalogFile); got != string(catalog) {
		t.Fatalf("catalog.md does not match the rendered catalog:\n%s", got)
	}
	if got := readOutput(t, root, IndexFile); !strings.Contains(got, "Octo-Labs/Daily-Notes") {
		t.Fatalf("index.json lost the descriptor casing:\n%s", got)
	}
	assertNoTempFiles(t, root)
}

func TestGenerateIsIdempotent(t *testing.T) {
	t.Parallel()
	root := newPopulatedRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	firstIndex := readOutput(t, root, IndexFile)
	firstCatalog := readOutput(t, root, CatalogFile)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	if readOutput(t, root, IndexFile) != firstIndex {
		t.Fatal("index.json changed between runs")
	}
	if readOutput(t, root, CatalogFile) != firstCatalog {
		t.Fatal("catalog.md changed between runs")
	}
}

func TestGenerateRepairsStaleOutputs(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	writeFile(t, filepath.Join(root, IndexFile), "{}\n")
	writeFile(t, filepath.Join(root, CatalogFile), "# stale\n")
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	if got := readOutput(t, root, IndexFile); got != emptyIndexJSON {
		t.Fatalf("index.json was not repaired: %q", got)
	}
	if got := readOutput(t, root, CatalogFile); got != emptyCatalogMD {
		t.Fatalf("catalog.md was not repaired: %q", got)
	}
}

func TestGenerateInvalidInputLeavesOutputsUnchanged(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	beforeIndex := readOutput(t, root, IndexFile)
	beforeCatalog := readOutput(t, root, CatalogFile)
	writeFile(t, filepath.Join(root, PackagesDir, "bad", "pkg.json"), "{}")
	if _, err := Generate(root); err == nil {
		t.Fatal("Generate accepted an invalid descriptor")
	}
	if readOutput(t, root, IndexFile) != beforeIndex {
		t.Fatal("index.json changed on invalid input")
	}
	if readOutput(t, root, CatalogFile) != beforeCatalog {
		t.Fatal("catalog.md changed on invalid input")
	}
	assertNoTempFiles(t, root)
}

func TestGenerateStageError(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o700); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := Generate(root); err == nil || !strings.Contains(err.Error(), "stage") {
		t.Fatalf("Generate error = %v, want a stage error", err)
	}
	if _, err := os.Stat(filepath.Join(root, IndexFile)); !os.IsNotExist(err) {
		t.Fatalf("index.json exists after a failed generate: %v", err)
	}
}

func TestGenerateRenameErrorOnIndex(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.MkdirAll(filepath.Join(root, IndexFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root); err == nil || !strings.Contains(err.Error(), "replace "+filepath.Join(root, IndexFile)) {
		t.Fatalf("Generate error = %v, want a replace error naming index.json", err)
	}
	assertNoTempFiles(t, root)
	if _, err := os.Stat(filepath.Join(root, CatalogFile)); !os.IsNotExist(err) {
		t.Fatalf("catalog.md exists after a failed index replace: %v", err)
	}
}

func TestGenerateRenameErrorOnCatalog(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if err := os.MkdirAll(filepath.Join(root, CatalogFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(root); err == nil || !strings.Contains(err.Error(), "replace "+filepath.Join(root, CatalogFile)) {
		t.Fatalf("Generate error = %v, want a replace error naming catalog.md", err)
	}
	assertNoTempFiles(t, root)
	if got := readOutput(t, root, IndexFile); got != emptyIndexJSON {
		t.Fatalf("index.json was not replaced before the catalog failure: %q", got)
	}
}

func TestCheckCurrent(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		root := newEmptyRepo(t)
		if _, err := Generate(root); err != nil {
			t.Fatal(err)
		}
		count, err := Check(root)
		if err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if count != 0 {
			t.Fatalf("Check count = %d, want 0", count)
		}
	})
	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		root := newPopulatedRepo(t)
		if _, err := Generate(root); err != nil {
			t.Fatal(err)
		}
		count, err := Check(root)
		if err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if count != 2 {
			t.Fatalf("Check count = %d, want 2", count)
		}
	})
}

func TestCheckStale(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	stale := "{},\n"
	if err := os.WriteFile(filepath.Join(root, IndexFile), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "index.json is stale") {
		t.Fatalf("Check error = %v, want a stale index.json problem", err)
	}
	if got := readOutput(t, root, IndexFile); got != stale {
		t.Fatal("Check rewrote the stale file")
	}
	if _, err := Check(root); err == nil || strings.Contains(err.Error(), "catalog.md") {
		t.Fatalf("Check error = %v, catalog.md must not be reported stale", err)
	}
}

func TestCheckMissing(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, CatalogFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "catalog.md is missing") {
		t.Fatalf("Check error = %v, want a missing catalog.md problem", err)
	}
	if _, err := os.Stat(filepath.Join(root, CatalogFile)); !os.IsNotExist(err) {
		t.Fatal("Check wrote the missing file")
	}
}

func TestCheckReportsBothProblems(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, IndexFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, CatalogFile), []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := func() error {
		_, err := Check(root)
		return err
	}()
	if err == nil {
		t.Fatal("Check accepted broken outputs")
	}
	if !strings.Contains(err.Error(), "index.json is missing") || !strings.Contains(err.Error(), "catalog.md is stale") {
		t.Fatalf("Check error = %v, want both problems", err)
	}
}

func TestCheckReportsReadErrors(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, IndexFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, IndexFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root); err == nil || !strings.Contains(err.Error(), "read "+filepath.Join(root, IndexFile)) {
		t.Fatalf("Check error = %v, want a read error naming index.json", err)
	}
}

func TestCheckInvalidDescriptorDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := newEmptyRepo(t)
	if _, err := Generate(root); err != nil {
		t.Fatal(err)
	}
	beforeIndex := readOutput(t, root, IndexFile)
	writeFile(t, filepath.Join(root, PackagesDir, "bad", "pkg.json"), "{")
	if _, err := Check(root); err == nil {
		t.Fatal("Check accepted an invalid descriptor")
	}
	if readOutput(t, root, IndexFile) != beforeIndex {
		t.Fatal("Check wrote during validation failure")
	}
}

func mustLoad(t *testing.T, root string) []Descriptor {
	t.Helper()
	descriptors, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	return descriptors
}
