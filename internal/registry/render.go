package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Record struct {
	Repository  string   `json:"repository"`
	Description string   `json:"description"`
	Maintainers []string `json:"maintainers"`
	Topics      []string `json:"topics"`
}

type indexDocument struct {
	SchemaVersion int      `json:"schemaVersion"`
	Packages      []Record `json:"packages"`
}

func BuildIndex(descriptors []Descriptor) []byte {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(indexDocument{SchemaVersion: SchemaVersion, Packages: sortedRecords(descriptors)})
	return buffer.Bytes()
}

func BuildCatalog(descriptors []Descriptor) ([]byte, error) {
	records := sortedRecords(descriptors)
	var catalog strings.Builder
	catalog.WriteString(catalogIntro)
	if len(records) == 0 {
		catalog.WriteString(catalogEmptyState)
		return []byte(catalog.String()), nil
	}
	for _, record := range records {
		owner, name, err := SplitRepository(record.Repository)
		if err != nil {
			return nil, fmt.Errorf("render catalog for %q: %w", record.Repository, err)
		}
		repositoryURL := fmt.Sprintf("https://github.com/%s/%s", owner, name)
		catalog.WriteString(fmt.Sprintf("- [%s](%s) ([tags](%s/tags)): %s\n", record.Repository, repositoryURL, repositoryURL, escapeMarkdown(record.Description)))
		catalog.WriteString(fmt.Sprintf("  - maintainers: %s\n", formatMaintainers(record.Maintainers)))
		if len(record.Topics) > 0 {
			catalog.WriteString(fmt.Sprintf("  - topics: %s\n", formatTopics(record.Topics)))
		}
	}
	return []byte(catalog.String()), nil
}

func sortedRecords(descriptors []Descriptor) []Record {
	records := make([]Record, 0, len(descriptors))
	for _, descriptor := range descriptors {
		records = append(records, Record{
			Repository:  descriptor.Repository,
			Description: descriptor.Description,
			Maintainers: sortedCopy(descriptor.Maintainers),
			Topics:      sortedCopy(descriptor.Topics),
		})
	}
	slices.SortFunc(records, compareRecords)
	return records
}

func compareRecords(a, b Record) int {
	if order := strings.Compare(strings.ToLower(a.Repository), strings.ToLower(b.Repository)); order != 0 {
		return order
	}
	return strings.Compare(a.Repository, b.Repository)
}

func sortedCopy(values []string) []string {
	copied := make([]string, len(values))
	copy(copied, values)
	slices.Sort(copied)
	return copied
}

func formatMaintainers(values []string) string {
	entries := make([]string, len(values))
	for index, value := range values {
		entries[index] = "@" + value
	}
	return strings.Join(entries, ", ")
}

func formatTopics(values []string) string {
	entries := make([]string, len(values))
	for index, value := range values {
		entries[index] = "`" + value + "`"
	}
	return strings.Join(entries, ", ")
}

func escapeMarkdown(value string) string {
	var escaped strings.Builder
	for index := 0; index < len(value); index++ {
		char := value[index]
		if isASCIIPunctuation(char) {
			escaped.WriteByte('\\')
		}
		escaped.WriteByte(char)
	}
	return escaped.String()
}

func isASCIIPunctuation(char byte) bool {
	return (char >= '!' && char <= '/') ||
		(char >= ':' && char <= '@') ||
		(char >= '[' && char <= '`') ||
		(char >= '{' && char <= '~')
}

const catalogIntro = "# Smidja package catalog\n" +
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
	"\n"

const catalogEmptyState = "No packages are listed yet. The registry starts empty on purpose; see the [contribution guide](.github/CONTRIBUTING.md) to submit the first descriptor.\n"
