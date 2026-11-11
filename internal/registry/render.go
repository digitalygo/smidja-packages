package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type Record struct {
	Repository   string            `json:"repository"`
	Description  string            `json:"description"`
	Maintainers  []string          `json:"maintainers"`
	Topics       []string          `json:"topics"`
	ID           string            `json:"id,omitempty"`
	Scopes       []string          `json:"scopes,omitempty"`
	ReleaseFeeds map[string]string `json:"releaseFeeds,omitempty"`
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
		if record.ID != "" {
			catalog.WriteString(fmt.Sprintf("  - package: `%s`; scopes: %s\n", escapeMarkdown(record.ID), formatTopics(record.Scopes)))
			targets := make([]string, 0, len(record.ReleaseFeeds))
			for target := range record.ReleaseFeeds {
				targets = append(targets, target)
			}
			slices.Sort(targets)
			for _, target := range targets {
				catalog.WriteString(fmt.Sprintf("  - %s release feed: %s\n", escapeMarkdown(target), escapeMarkdown(record.ReleaseFeeds[target])))
			}
		}
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
			ID:          descriptor.ID, Scopes: sortedCopy(descriptor.Scopes), ReleaseFeeds: descriptor.ReleaseFeeds,
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

const catalogIntro = `# Smidja package catalog

Generated discovery index of [Smidja](https://github.com/digitalygo/smidja) runtime packages. Listing is not endorsement. Packages may contain executable tools; inspect and trust the publisher before installing. This registry neither hosts packages nor verifies their runtime contents.

Choose a release feed for your exact OS/architecture and configure that HTTPS source in the manager or client. The current runtime does not automatically select targets or install from this catalog. Publisher feeds, not this index, supply versions and archive checksums. Private packages stay outside this public catalog.

Submission rules: [contribution guide](.github/CONTRIBUTING.md). Schema: [registry format](docs/registry-format.md).

## Packages

`

const catalogEmptyState = "No packages are listed yet. The registry starts empty on purpose; see the [contribution guide](.github/CONTRIBUTING.md) to submit the first descriptor.\n"
