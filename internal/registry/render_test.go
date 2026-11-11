package registry

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildIndexEmpty(t *testing.T) {
	t.Parallel()
	for _, descriptors := range [][]Descriptor{nil, {}} {
		got := BuildIndex(descriptors)
		if string(got) != emptyIndexJSON {
			t.Fatalf("BuildIndex = %q, want %q", got, emptyIndexJSON)
		}
	}
}

func TestBuildIndexPopulated(t *testing.T) {
	t.Parallel()
	want := "{\n" +
		"  \"schemaVersion\": 2,\n" +
		"  \"packages\": [\n" +
		"    {\n" +
		"      \"repository\": \"acme/tools\",\n" +
		"      \"description\": \"A set of review prompts.\",\n" +
		"      \"maintainers\": [\n" +
		"        \"alice\",\n" +
		"        \"bob\"\n" +
		"      ],\n" +
		"      \"topics\": []\n" +
		"    },\n" +
		"    {\n" +
		"      \"repository\": \"Octo-Labs/Daily-Notes\",\n" +
		"      \"description\": \"Daily notes skill with config defaults.\",\n" +
		"      \"maintainers\": [\n" +
		"        \"alice\"\n" +
		"      ],\n" +
		"      \"topics\": [\n" +
		"        \"daily\",\n" +
		"        \"notes\"\n" +
		"      ]\n" +
		"    }\n" +
		"  ]\n" +
		"}\n"
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "Octo-Labs/Daily-Notes", Description: "Daily notes skill with config defaults.", Maintainers: []string{"alice"}, Topics: []string{"notes", "daily"}},
		{SchemaVersion: 2, Repository: "acme/tools", Description: "A set of review prompts.", Maintainers: []string{"bob", "alice"}},
	}
	got := BuildIndex(descriptors)
	if string(got) != want {
		t.Fatalf("BuildIndex mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildIndexSortsCaseInsensitively(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "Zeta/Widget", Description: "d", Maintainers: []string{"a"}},
		{SchemaVersion: 2, Repository: "alpha/tools", Description: "d", Maintainers: []string{"a"}},
		{SchemaVersion: 2, Repository: "Beta/Gamma", Description: "d", Maintainers: []string{"a"}},
	}
	index := string(BuildIndex(descriptors))
	alpha := strings.Index(index, `"alpha/tools"`)
	beta := strings.Index(index, `"Beta/Gamma"`)
	zeta := strings.Index(index, `"Zeta/Widget"`)
	if !(0 <= alpha && alpha < beta && beta < zeta) {
		t.Fatalf("records are not sorted case insensitively:\n%s", index)
	}
}

func TestBuildIndexTieBreaksOnExactRepository(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "alpha/b", Description: "d", Maintainers: []string{"a"}},
		{SchemaVersion: 2, Repository: "Alpha/b", Description: "d", Maintainers: []string{"a"}},
	}
	index := string(BuildIndex(descriptors))
	if strings.Index(index, `"Alpha/b"`) > strings.Index(index, `"alpha/b"`) {
		t.Fatalf("tie break is wrong:\n%s", index)
	}
}

func TestBuildIndexSourceOrderIndependence(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "acme/tools", Description: "d1", Maintainers: []string{"a"}, Topics: []string{"x"}},
		{SchemaVersion: 2, Repository: "Octo-Labs/Daily-Notes", Description: "d2", Maintainers: []string{"b"}},
		{SchemaVersion: 2, Repository: "mid/marker", Description: "d3", Maintainers: []string{"c"}, Topics: []string{"y", "x"}},
	}
	first := string(BuildIndex(descriptors))
	reversed := slices.Clone(descriptors)
	slices.Reverse(reversed)
	second := string(BuildIndex(reversed))
	if first != second {
		t.Fatalf("index depends on source order:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestBuildIndexEscapesHTMLSignificantCharacters(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "a/b", Description: "<b>bold</b> & \"quoted\" <script>alert(1)</script>", Maintainers: []string{"a"}},
	}
	index := string(BuildIndex(descriptors))
	if strings.Contains(index, "<script>") || strings.Contains(index, "<b>") {
		t.Fatalf("index contains raw HTML significant characters:\n%s", index)
	}
	if !strings.Contains(index, `\u003cscript\u003ealert(1)\u003c/script\u003e`) {
		t.Fatalf("index does not escape script tags:\n%s", index)
	}
	if !strings.Contains(index, `\u0026`) {
		t.Fatalf("index does not escape the ampersand:\n%s", index)
	}
}

func TestBuildCatalogEmpty(t *testing.T) {
	t.Parallel()
	for _, descriptors := range [][]Descriptor{nil, {}} {
		got, err := BuildCatalog(descriptors)
		if err != nil {
			t.Fatalf("BuildCatalog returned error: %v", err)
		}
		if string(got) != emptyCatalogMD {
			t.Fatalf("BuildCatalog = %q, want %q", got, emptyCatalogMD)
		}
	}
}

func TestBuildCatalogPopulated(t *testing.T) {
	t.Parallel()
	want := emptyCatalogMD[:strings.Index(emptyCatalogMD, "No packages are listed")] +
		"- [acme/tools](https://github.com/acme/tools) ([tags](https://github.com/acme/tools/tags)): A set of review prompts\\.\n" +
		"  - maintainers: @alice, @bob\n" +
		"- [Octo-Labs/Daily-Notes](https://github.com/Octo-Labs/Daily-Notes) ([tags](https://github.com/Octo-Labs/Daily-Notes/tags)): Daily notes skill with config defaults\\.\n" +
		"  - maintainers: @alice\n" +
		"  - topics: `daily`, `notes`\n"
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "Octo-Labs/Daily-Notes", Description: "Daily notes skill with config defaults.", Maintainers: []string{"alice"}, Topics: []string{"notes", "daily"}},
		{SchemaVersion: 2, Repository: "acme/tools", Description: "A set of review prompts.", Maintainers: []string{"bob", "alice"}},
	}
	got, err := BuildCatalog(descriptors)
	if err != nil {
		t.Fatalf("BuildCatalog returned error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("BuildCatalog mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildCatalogRejectsInvalidRepository(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{{SchemaVersion: 2, Repository: "not-a-repo", Description: "d", Maintainers: []string{"a"}}}
	if _, err := BuildCatalog(descriptors); err == nil || !strings.Contains(err.Error(), "render catalog") {
		t.Fatalf("BuildCatalog error = %v, want a render error", err)
	}
}

func TestBuildCatalogEscapesMarkdownInjection(t *testing.T) {
	t.Parallel()
	injection := "[x](https://evil.example) <img src=x onerror=alert(1)> `cmd` *bold* _em_ & quot $math$ |pipe| ~strike~ #head !bang (paren) {brace} back\\slash"
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "evil/pkg", Description: injection, Maintainers: []string{"a"}},
	}
	catalog, err := BuildCatalog(descriptors)
	if err != nil {
		t.Fatalf("BuildCatalog returned error: %v", err)
	}
	rendered := string(catalog)
	_, descriptionPart, found := strings.Cut(rendered, "/tags")
	if !found {
		t.Fatalf("catalog does not contain the entry line:\n%s", rendered)
	}
	for _, banned := range []string{
		"](https://evil.example)",
		"`cmd`",
		"*bold*",
		"_em_",
		"$math$",
		"|pipe|",
		"~strike~",
		"(paren)",
		"{brace}",
		"back\\slash",
	} {
		if strings.Contains(descriptionPart, banned) {
			t.Fatalf("catalog description contains an unescaped %q:\n%s", banned, rendered)
		}
	}
	if !strings.Contains(descriptionPart, escapeMarkdown(injection)) {
		t.Fatalf("catalog does not contain the fully escaped description:\n%s", rendered)
	}
	for _, escapedForm := range []string{"\\[x\\]\\(https\\:\\/\\/evil\\.example\\)", "\\<img", "\\`cmd\\`", "\\#head", "\\!bang", "\\(paren\\)", "\\{brace\\}", "\\& quot", "back\\\\slash"} {
		if !strings.Contains(descriptionPart, escapedForm) {
			t.Fatalf("catalog description misses the escaped form %q:\n%s", escapedForm, rendered)
		}
	}
}

func TestBuildCatalogPreservesUnicodeInDescriptions(t *testing.T) {
	t.Parallel()
	descriptors := []Descriptor{
		{SchemaVersion: 2, Repository: "a/b", Description: "每日笔记 émoji 🎉", Maintainers: []string{"a"}},
	}
	catalog, err := BuildCatalog(descriptors)
	if err != nil {
		t.Fatalf("BuildCatalog returned error: %v", err)
	}
	if !strings.Contains(string(catalog), "每日笔记 émoji 🎉") {
		t.Fatalf("catalog mangles Unicode:\n%s", catalog)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	t.Parallel()
	punctuation := "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	escaped := escapeMarkdown(punctuation)
	var want strings.Builder
	for _, char := range []byte(punctuation) {
		want.WriteByte('\\')
		want.WriteByte(char)
	}
	if escaped != want.String() {
		t.Fatalf("escapeMarkdown = %q, want %q", escaped, want.String())
	}
	if got := escapeMarkdown("aBc 123\n\t ré Kitchen 中"); got != "aBc 123\n\t ré Kitchen 中" {
		t.Fatalf("escapeMarkdown changed non punctuation text: %q", got)
	}
	if got := escapeMarkdown("a\"b"); got != "a\\\"b" {
		t.Fatalf("escapeMarkdown quote handling = %q", got)
	}
}

func TestGeneratedOutputsEndWithSingleNewline(t *testing.T) {
	t.Parallel()
	index := BuildIndex(nil)
	if !strings.HasSuffix(string(index), "}\n") || strings.HasSuffix(string(index), "\n\n") {
		t.Fatalf("index.json does not end with exactly one newline: %q", index)
	}
	catalog, err := BuildCatalog(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(catalog), "descriptor.\n") || strings.HasSuffix(string(catalog), "\n\n") {
		t.Fatalf("catalog.md does not end with exactly one newline: %q", catalog)
	}
}
