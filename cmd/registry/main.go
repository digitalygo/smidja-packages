package main

import (
	"fmt"
	"io"
	"os"

	"github.com/digitalygo/smidja-packages/internal/registry"
)

const usage = "usage: go run ./cmd/registry <command>\n" +
	"\n" +
	"commands:\n" +
	"  generate    validate every descriptor and write index.json and catalog.md\n" +
	"  check       validate every descriptor and verify the committed outputs are exact\n" +
	"  help        show this help\n"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "generate", "check":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "registry: command %q takes no arguments\n\n%s", args[0], usage)
			return 2
		}
		root, err := registry.FindRoot(".")
		if err != nil {
			fmt.Fprintf(stderr, "registry: %v\n", err)
			return 1
		}
		if args[0] == "generate" {
			count, err := registry.Generate(root)
			if err != nil {
				fmt.Fprintf(stderr, "registry: %v\n", err)
				return 1
			}
			fmt.Fprintf(stdout, "wrote %s and %s for %d %s\n", registry.IndexFile, registry.CatalogFile, count, packageWord(count))
			return 0
		}
		count, err := registry.Check(root)
		if err != nil {
			fmt.Fprintf(stderr, "registry: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s and %s are up to date (%d %s)\n", registry.IndexFile, registry.CatalogFile, count, packageWord(count))
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "registry: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func packageWord(count int) string {
	if count == 1 {
		return "package"
	}
	return "packages"
}
