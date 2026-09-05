package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func newCommandRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/digitalygo/smidja-packages\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", ".gitkeep"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runCommand(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestRunHelp(t *testing.T) {
	t.Parallel()
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()
			code, stdout, stderr := runCommand(t, arg)
			if code != 0 {
				t.Fatalf("exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout, "usage: go run ./cmd/registry <command>") {
				t.Fatalf("stdout = %q, want usage", stdout)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRunNoArgs(t *testing.T) {
	t.Parallel()
	code, stdout, stderr := runCommand(t)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "usage: go run ./cmd/registry <command>") {
		t.Fatalf("stderr = %q, want usage", stderr)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()
	code, _, stderr := runCommand(t, "frobnicate")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, `unknown command "frobnicate"`) {
		t.Fatalf("stderr = %q, want an unknown command message", stderr)
	}
}

func TestRunExtraArguments(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"generate", "check"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			code, _, stderr := runCommand(t, command, "extra")
			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr, "takes no arguments") {
				t.Fatalf("stderr = %q, want an argument error", stderr)
			}
		})
	}
}

func TestRunGenerateSuccess(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	code, stdout, stderr := runCommand(t, "generate")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "wrote index.json and catalog.md for 0 packages") {
		t.Fatalf("stdout = %q", stdout)
	}
	index, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(index) != emptyIndexJSON {
		t.Fatalf("index.json = %q, want %q", index, emptyIndexJSON)
	}
	catalog, err := os.ReadFile(filepath.Join(root, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(catalog) != emptyCatalogMD {
		t.Fatalf("catalog.md = %q, want the generated empty catalog", catalog)
	}
}

func TestRunGenerateSinglePackage(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	if err := os.Remove(filepath.Join(root, "packages", ".gitkeep")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	descriptor := `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["alice"]}`
	if err := os.WriteFile(filepath.Join(root, "packages", "a", "b.json"), []byte(descriptor), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCommand(t, "generate")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "wrote index.json and catalog.md for 1 package") {
		t.Fatalf("stdout = %q", stdout)
	}
	index, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `"repository": "a/b"`) {
		t.Fatalf("index.json = %q", index)
	}
}

func TestRunCheckSuccess(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	if code, _, stderr := runCommand(t, "generate"); code != 0 {
		t.Fatalf("generate exit code = %d, stderr = %s", code, stderr)
	}
	code, stdout, stderr := runCommand(t, "check")
	if code != 0 {
		t.Fatalf("check exit code = %d, stderr = %s", code, stderr)
	}
	if !strings.Contains(stdout, "index.json and catalog.md are up to date (0 packages)") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestRunCheckStale(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	if code, _, stderr := runCommand(t, "generate"); code != 0 {
		t.Fatalf("generate exit code = %d, stderr = %s", code, stderr)
	}
	stale := "{},\n"
	if err := os.WriteFile(filepath.Join(root, "index.json"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCommand(t, "check")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "index.json is stale") {
		t.Fatalf("stderr = %q, want a stale report", stderr)
	}
	got, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != stale {
		t.Fatal("check rewrote the stale file")
	}
}

func TestRunCheckMissing(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	if code, _, stderr := runCommand(t, "generate"); code != 0 {
		t.Fatalf("generate exit code = %d, stderr = %s", code, stderr)
	}
	if err := os.Remove(filepath.Join(root, "catalog.md")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCommand(t, "check")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "catalog.md is missing") {
		t.Fatalf("stderr = %q, want a missing report", stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "catalog.md")); !os.IsNotExist(err) {
		t.Fatal("check wrote the missing file")
	}
}

func TestRunGenerateInvalidLeavesOutputs(t *testing.T) {
	root := newCommandRepo(t)
	t.Chdir(root)
	if code, _, stderr := runCommand(t, "generate"); code != 0 {
		t.Fatalf("generate exit code = %d, stderr = %s", code, stderr)
	}
	beforeIndex, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	beforeCatalog, err := os.ReadFile(filepath.Join(root, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", "bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", "bad", "pkg.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCommand(t, "generate")
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "packages/bad/pkg.json") {
		t.Fatalf("stderr = %q, want the descriptor path", stderr)
	}
	afterIndex, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	afterCatalog, err := os.ReadFile(filepath.Join(root, "catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterIndex) != string(beforeIndex) || string(afterCatalog) != string(beforeCatalog) {
		t.Fatal("generate changed the outputs on invalid input")
	}
}

func TestRunOutsideRepository(t *testing.T) {
	for _, command := range []string{"generate", "check"} {
		t.Run(command, func(t *testing.T) {
			t.Chdir(t.TempDir())
			code, _, stderr := runCommand(t, command)
			if code != 1 {
				t.Fatalf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr, "could not find the repository root") {
				t.Fatalf("stderr = %q, want a root not found message", stderr)
			}
		})
	}
}

func TestEndToEndGoRun(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine the command directory")
	}
	commandDir := filepath.Dir(thisFile)
	binary := filepath.Join(t.TempDir(), "registry")
	build := exec.Command("go", "build", "-o", binary, commandDir)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v, output: %s", err, output)
	}
	root := newCommandRepo(t)
	runBinary := func(args ...string) (string, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = root
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("go run %v failed: %v, stderr: %s", args, err, stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	stdout, _ := runBinary("generate")
	if !strings.Contains(stdout, "wrote index.json and catalog.md for 0 packages") {
		t.Fatalf("stdout = %q", stdout)
	}
	index, err := os.ReadFile(filepath.Join(root, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(index) != emptyIndexJSON {
		t.Fatalf("index.json = %q, want %q", index, emptyIndexJSON)
	}
	stdout, _ = runBinary("check")
	if !strings.Contains(stdout, "index.json and catalog.md are up to date (0 packages)") {
		t.Fatalf("stdout = %q", stdout)
	}
	failed := exec.Command(binary, "frobnicate")
	failed.Dir = root
	var stderr bytes.Buffer
	failed.Stderr = &stderr
	if err := failed.Run(); err == nil {
		t.Fatal("go run with an unknown command must fail")
	} else if !strings.Contains(stderr.String(), `unknown command "frobnicate"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
