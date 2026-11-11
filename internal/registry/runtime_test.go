package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimeDescriptor(t *testing.T) {
	raw := `{"schemaVersion":2,"repository":"example/tools","description":"Runtime tools","maintainers":["alice"],"id":"example.tools","scopes":["client","manager"],"releaseFeeds":{"linux/amd64":"https://example.org/linux-amd64/release.json","linux/arm64":"https://example.org/linux-arm64/release.json","darwin/amd64":"https://example.org/darwin-amd64/release.json","darwin/arm64":"https://example.org/darwin-arm64/release.json"}}`
	descriptor, err := ParseDescriptor([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	index := BuildIndex([]Descriptor{descriptor})
	var output indexDocument
	if err := json.Unmarshal(index, &output); err != nil {
		t.Fatal(err)
	}
	if output.SchemaVersion != 2 || len(output.Packages) != 1 || output.Packages[0].ID != "example.tools" || len(output.Packages[0].ReleaseFeeds) != 4 {
		t.Fatalf("lost metadata: %s", index)
	}
	catalog, err := BuildCatalog([]Descriptor{descriptor})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(catalog), "package: `example\\.tools`") || !strings.Contains(string(catalog), "darwin\\/arm64 release feed:") {
		t.Fatalf("missing discovery metadata: %s", catalog)
	}
	if strings.Index(string(catalog), "darwin\\/amd64 release feed:") > strings.Index(string(catalog), "linux\\/arm64 release feed:") {
		t.Fatal("feeds not sorted")
	}
	for _, field := range []string{"id", "scopes", "releaseFeeds"} {
		var fields map[string]any
		json.Unmarshal([]byte(raw), &fields)
		delete(fields, field)
		data, _ := json.Marshal(fields)
		if _, err := ParseDescriptor(data); err == nil || !strings.Contains(err.Error(), "missing required field") {
			t.Fatalf("missing %s: %v", field, err)
		}
	}
	tests := []struct {
		field string
		value any
	}{
		{"id", "UPPER"}, {"id", ""}, {"id", strings.Repeat("x", 129)}, {"id", nil},
		{"scopes", []string{}}, {"scopes", []string{"manager", "manager"}}, {"scopes", []string{"other"}}, {"scopes", []string{"manager", "client", "other"}},
		{"releaseFeeds", map[string]string{}}, {"releaseFeeds", nil}, {"releaseFeeds", []string{}},
		{"releaseFeeds", map[string]string{"linux/armv7": "https://example.org/r"}},
		{"releaseFeeds", map[string]string{"a": "https://example.org/r", "b": "https://example.org/r", "c": "https://example.org/r", "d": "https://example.org/r", "e": "https://example.org/r"}},
	}
	for _, feed := range []string{"http://example.org/r", "https:///r", "https://user:pass@example.org/r", "https://example.org/r?token=x", "https://example.org/r?", "https://example.org/r#x", "https://example.org/r#", "https://example.org/r x", "https://example.org/%zz", "https://example.org/`x", "https://example.org/\\x"} {
		tests = append(tests, struct {
			field string
			value any
		}{"releaseFeeds", map[string]string{"linux/amd64": feed}})
	}
	for _, test := range tests {
		var fields map[string]any
		json.Unmarshal([]byte(raw), &fields)
		fields[test.field] = test.value
		data, _ := json.Marshal(fields)
		if _, err := ParseDescriptor(data); err == nil {
			t.Fatalf("accepted %s=%v", test.field, test.value)
		}
	}
}

func TestReleaseFeedParsingErrors(t *testing.T) {
	for _, raw := range []string{`{"schemaVersion":1}`, `{"releaseFeeds":{"linux/amd64":"a","linux\u002famd64":"b"}}`, `{"releaseFeeds":`, `{"releaseFeeds":{"linux/amd64":`, `{"releaseFeeds":{"linux/amd64":"https://example.org/r",`, `{"releaseFeeds":{"linux/amd64":"https://example.org/r"`, `{"releaseFeeds":{"linux/amd64":"a","linux/amd64":"b"}}`, `{"releaseFeeds":{"linux/amd64":null}}`, `{"releaseFeeds":{"linux/amd64":2}}`} {
		if _, err := ParseDescriptor([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
