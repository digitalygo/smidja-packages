package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func marshalDescriptor(t *testing.T, fields map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	return data
}

func TestParseDescriptorValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		data       string
		repository string
	}{
		{"minimal", `{` + validDescriptorBody + `}`, "a/b"},
		{"with topics", `{"schemaVersion":1,"repository":"Owner/Repo.Name","description":"A test package.","maintainers":["a-b","u2"],"topics":["t1","t-two"]}`, "Owner/Repo.Name"},
		{"empty topics", `{"schemaVersion":1,"repository":"a/b","description":"A test package.","maintainers":["u"],"topics":[]}`, "a/b"},
		{"trailing whitespace", "{\n  " + validDescriptorBody + "\n}\n", "a/b"},
		{"unicode description", `{"schemaVersion":1,"repository":"a/b","description":"每日笔记 émoji 🎉 review prompts","maintainers":["u"]}`, "a/b"},
		{"html in description", `{"schemaVersion":1,"repository":"a/b","description":"<b>bold</b> & \"quoted\"","maintainers":["u"]}`, "a/b"},
		{"digits owner", `{"schemaVersion":1,"repository":"123/tool-s","description":"d","maintainers":["u"]}`, "123/tool-s"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			descriptor, err := ParseDescriptor([]byte(test.data))
			if err != nil {
				t.Fatalf("ParseDescriptor returned error: %v", err)
			}
			if descriptor.SchemaVersion != 1 {
				t.Errorf("schemaVersion = %d, want 1", descriptor.SchemaVersion)
			}
			if descriptor.Repository != test.repository {
				t.Errorf("repository = %q, want %q", descriptor.Repository, test.repository)
			}
		})
	}
}

func TestParseDescriptorErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{"empty file", "", "invalid JSON"},
		{"whitespace only", "  \n\t ", "invalid JSON"},
		{"byte order mark", "\ufeff{" + validDescriptorBody + "}", "invalid JSON"},
		{"top level array", "[]", "expected"},
		{"top level string", `"x"`, "expected"},
		{"top level number", "1", "expected"},
		{"top level null", "null", "expected"},
		{"unclosed object", `{"schemaVersion":1`, "invalid JSON"},
		{"trailing comma", `{"schemaVersion":1,}`, "invalid JSON"},
		{"trailing object", `{` + validDescriptorBody + `} {}`, "trailing data"},
		{"trailing garbage", `{` + validDescriptorBody + `}x`, "invalid JSON"},
		{"trailing array", `{` + validDescriptorBody + `} []`, "trailing data"},
		{"trailing number", `{` + validDescriptorBody + `} 1`, "trailing data"},
		{"duplicate schemaVersion", `{"schemaVersion":1,"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"]}`, `duplicate field "schemaVersion"`},
		{"escaped duplicate schemaVersion", `{"\u0073chemaVersion":1,"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"]}`, `duplicate field "schemaVersion"`},
		{"escaped duplicate repository", `{"repository":"a/b","\u0072epository":"a/b","schemaVersion":1,"description":"d","maintainers":["u"]}`, `duplicate field "repository"`},
		{"partially escaped duplicate", `{"maintainers":["u"],"maintain\u0065rs":["u"],"schemaVersion":1,"repository":"a/b","description":"d"}`, `duplicate field "maintainers"`},
		{"unknown field", `{` + validDescriptorBody + `,"extra":1}`, `unknown field "extra"`},
		{"case aliased schemaVersion", `{"SchemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"]}`, `unknown field "SchemaVersion"`},
		{"case aliased repository", `{"REPOSITORY":"a/b","schemaVersion":1,"description":"d","maintainers":["u"]}`, `unknown field "REPOSITORY"`},
		{"case aliased maintainers", `{"Maintainers":["u"],"schemaVersion":1,"repository":"a/b","description":"d"}`, `unknown field "Maintainers"`},
		{"missing schemaVersion", `{"repository":"a/b","description":"d","maintainers":["u"]}`, `missing required field "schemaVersion"`},
		{"missing repository", `{"schemaVersion":1,"description":"d","maintainers":["u"]}`, `missing required field "repository"`},
		{"missing description", `{"schemaVersion":1,"repository":"a/b","maintainers":["u"]}`, `missing required field "description"`},
		{"missing maintainers", `{"schemaVersion":1,"repository":"a/b","description":"d"}`, `missing required field "maintainers"`},
		{"schemaVersion zero", `{"schemaVersion":0,"repository":"a/b","description":"d","maintainers":["u"]}`, "unsupported schemaVersion 0"},
		{"schemaVersion two", `{"schemaVersion":2,"repository":"a/b","description":"d","maintainers":["u"]}`, "unsupported schemaVersion 2"},
		{"schemaVersion negative", `{"schemaVersion":-1,"repository":"a/b","description":"d","maintainers":["u"]}`, "unsupported schemaVersion -1"},
		{"schemaVersion string", `{"schemaVersion":"1","repository":"a/b","description":"d","maintainers":["u"]}`, "must be a number"},
		{"schemaVersion float", `{"schemaVersion":1.0,"repository":"a/b","description":"d","maintainers":["u"]}`, "must be the whole number 1"},
		{"schemaVersion exponent", `{"schemaVersion":1e0,"repository":"a/b","description":"d","maintainers":["u"]}`, "must be the whole number 1"},
		{"schemaVersion bool", `{"schemaVersion":true,"repository":"a/b","description":"d","maintainers":["u"]}`, "must be a number"},
		{"schemaVersion null", `{"schemaVersion":null,"repository":"a/b","description":"d","maintainers":["u"]}`, "must not be null"},
		{"repository null", `{"schemaVersion":1,"repository":null,"description":"d","maintainers":["u"]}`, "must not be null"},
		{"repository number", `{"schemaVersion":1,"repository":1,"description":"d","maintainers":["u"]}`, "must be a string"},
		{"repository array", `{"schemaVersion":1,"repository":[],"description":"d","maintainers":["u"]}`, "must be a string"},
		{"repository object", `{"schemaVersion":1,"repository":{},"description":"d","maintainers":["u"]}`, "must be a string"},
		{"repository bool", `{"schemaVersion":1,"repository":true,"description":"d","maintainers":["u"]}`, "must be a string"},
		{"repository no slash", `{"schemaVersion":1,"repository":"ab","description":"d","maintainers":["u"]}`, "must be owner/repo"},
		{"repository two slashes", `{"schemaVersion":1,"repository":"a/b/c","description":"d","maintainers":["u"]}`, "must be owner/repo"},
		{"repository empty owner", `{"schemaVersion":1,"repository":"/b","description":"d","maintainers":["u"]}`, "must be owner/repo"},
		{"repository empty name", `{"schemaVersion":1,"repository":"a/","description":"d","maintainers":["u"]}`, "must be owner/repo"},
		{"repository url", `{"schemaVersion":1,"repository":"https://github.com/a/b","description":"d","maintainers":["u"]}`, "must be owner/repo"},
		{"repository version suffix", `{"schemaVersion":1,"repository":"a/b@v1.0.0","description":"d","maintainers":["u"]}`, "must only contain ASCII letters, digits, underscores, hyphens, and dots"},
		{"repository query", `{"schemaVersion":1,"repository":"a/b?x=1","description":"d","maintainers":["u"]}`, "must only contain ASCII letters, digits, underscores, hyphens, and dots"},
		{"repository fragment", `{"schemaVersion":1,"repository":"a/b#c","description":"d","maintainers":["u"]}`, "must only contain ASCII letters, digits, underscores, hyphens, and dots"},
		{"repository owner leading hyphen", `{"schemaVersion":1,"repository":"-a/b","description":"d","maintainers":["u"]}`, "must be alphanumeric segments separated by single hyphens"},
		{"repository owner trailing hyphen", `{"schemaVersion":1,"repository":"a-/b","description":"d","maintainers":["u"]}`, "must be alphanumeric segments separated by single hyphens"},
		{"repository owner double hyphen", `{"schemaVersion":1,"repository":"a--b/c","description":"d","maintainers":["u"]}`, "must be alphanumeric segments separated by single hyphens"},
		{"repository owner underscore", `{"schemaVersion":1,"repository":"a_b/c","description":"d","maintainers":["u"]}`, "must be alphanumeric segments separated by single hyphens"},
		{"repository owner too long", `{"schemaVersion":1,"repository":"` + strings.Repeat("o", 40) + `/b","description":"d","maintainers":["u"]}`, "longer than 39 characters"},
		{"repository name dot", `{"schemaVersion":1,"repository":"a/.","description":"d","maintainers":["u"]}`, "is not allowed"},
		{"repository name dot dot", `{"schemaVersion":1,"repository":"a/..","description":"d","maintainers":["u"]}`, "is not allowed"},
		{"repository name too long", `{"schemaVersion":1,"repository":"a/` + strings.Repeat("r", 101) + `","description":"d","maintainers":["u"]}`, "longer than 100 characters"},
		{"repository name bad char", `{"schemaVersion":1,"repository":"a/b~c","description":"d","maintainers":["u"]}`, "must only contain ASCII letters, digits, underscores, hyphens, and dots"},
		{"repository name tilde is rejected by path check later", `{"schemaVersion":1,"repository":"a/b c","description":"d","maintainers":["u"]}`, "must only contain ASCII letters, digits, underscores, hyphens, and dots"},
		{"description empty", `{"schemaVersion":1,"repository":"a/b","description":"","maintainers":["u"]}`, "is required"},
		{"description untrimmed left", `{"schemaVersion":1,"repository":"a/b","description":" x","maintainers":["u"]}`, "leading or trailing whitespace"},
		{"description untrimmed right", `{"schemaVersion":1,"repository":"a/b","description":"x ","maintainers":["u"]}`, "leading or trailing whitespace"},
		{"description tab edges", `{"schemaVersion":1,"repository":"a/b","description":"\tx\t","maintainers":["u"]}`, "leading or trailing whitespace"},
		{"description nbsp edge", `{"schemaVersion":1,"repository":"a/b","description":" x","maintainers":["u"]}`, "leading or trailing whitespace"},
		{"description newline escape", `{"schemaVersion":1,"repository":"a/b","description":"a\nb","maintainers":["u"]}`, "control characters"},
		{"description tab escape", `{"schemaVersion":1,"repository":"a/b","description":"a\tb","maintainers":["u"]}`, "control characters"},
		{"description null escape", `{"schemaVersion":1,"repository":"a/b","description":"a\u0000b","maintainers":["u"]}`, "control characters"},
		{"description unit separator escape", `{"schemaVersion":1,"repository":"a/b","description":"a\u001fb","maintainers":["u"]}`, "control characters"},
		{"description delete escape", `{"schemaVersion":1,"repository":"a/b","description":"a\u007fb","maintainers":["u"]}`, "control characters"},
		{"description line separator", `{"schemaVersion":1,"repository":"a/b","description":"a\u2028b","maintainers":["u"]}`, "single line"},
		{"description paragraph separator", `{"schemaVersion":1,"repository":"a/b","description":"a\u2029b","maintainers":["u"]}`, "single line"},
		{"description zero width space", `{"schemaVersion":1,"repository":"a/b","description":"a\u200bb","maintainers":["u"]}`, "format characters"},
		{"description zero width no-break space", `{"schemaVersion":1,"repository":"a/b","description":"a\ufeffb","maintainers":["u"]}`, "format characters"},
		{"description word joiner", `{"schemaVersion":1,"repository":"a/b","description":"a\u2060b","maintainers":["u"]}`, "format characters"},
		{"description left to right embed", `{"schemaVersion":1,"repository":"a/b","description":"a\u202ab","maintainers":["u"]}`, "format characters"},
		{"description left to right override", `{"schemaVersion":1,"repository":"a/b","description":"a\u202db","maintainers":["u"]}`, "format characters"},
		{"description right to left override", `{"schemaVersion":1,"repository":"a/b","description":"a\u202eb","maintainers":["u"]}`, "format characters"},
		{"description left to right isolate", `{"schemaVersion":1,"repository":"a/b","description":"a\u2066b","maintainers":["u"]}`, "format characters"},
		{"description pop directional isolate", `{"schemaVersion":1,"repository":"a/b","description":"a\u2069b","maintainers":["u"]}`, "format characters"},
		{"description arabic letter mark", `{"schemaVersion":1,"repository":"a/b","description":"a\u061cb","maintainers":["u"]}`, "format characters"},
		{"description null", `{"schemaVersion":1,"repository":"a/b","description":null,"maintainers":["u"]}`, "must not be null"},
		{"description number", `{"schemaVersion":1,"repository":"a/b","description":1,"maintainers":["u"]}`, "must be a string"},
		{"maintainers empty array", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[]}`, "requires between 1 and 10"},
		{"maintainers null", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":null}`, "must not be null"},
		{"maintainers string", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":"u"}`, "must be an array"},
		{"maintainers object", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":{}}`, "must be an array"},
		{"maintainers eleven", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u1","u2","u3","u4","u5","u6","u7","u8","u9","u10","u11"]}`, "requires between 1 and 10"},
		{"maintainers duplicate", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u","u"]}`, "duplicate entry"},
		{"maintainers uppercase", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["Alice"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers empty entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[""]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers underscore", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["a_b"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers leading hyphen", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["-a"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers trailing hyphen", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["a-"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers double hyphen", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["a--b"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"maintainers too long", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["` + strings.Repeat("m", 40) + `"]}`, "longer than 39 characters"},
		{"maintainers null entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[null]}`, "entries must not be null"},
		{"maintainers number entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[1]}`, "entries must be strings"},
		{"maintainers object entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[{}]}`, "entries must be strings"},
		{"maintainers array entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":[[]]}`, "entries must be strings"},
		{"topics null", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":null}`, "must not be null"},
		{"topics string", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":"t"}`, "must be an array"},
		{"topics six", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["t1","t2","t3","t4","t5","t6"]}`, "requires between 0 and 5"},
		{"topics duplicate", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["t","t"]}`, "duplicate entry"},
		{"topics uppercase", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["T"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"topics underscore", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["a_b"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"topics leading hyphen", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["-t"]}`, "must be lowercase letters and digits separated by single hyphens"},
		{"topics too long", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":["` + strings.Repeat("t", 36) + `"]}`, "longer than 35 characters"},
		{"topics null entry", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"],"topics":[null]}`, "entries must not be null"},
		{"invalid escape", `{"schemaVersion":1,"repository":"a/b","description":"d\qxyz","maintainers":["u"]}`, "invalid JSON"},
		{"unclosed object at close delimiter", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"]`, "invalid JSON"},
		{"schemaVersion token error", `{"schemaVersion":`, "invalid JSON"},
		{"maintainers token error", `{"maintainers":`, "invalid JSON"},
		{"maintainers element token error", `{"maintainers":["u",`, "invalid JSON"},
		{"maintainers unclosed array", `{"maintainers":["u"`, "invalid JSON"},
		{"object closed with array delimiter", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"]]`, "invalid JSON"},
		{"array closed with object delimiter", `{"schemaVersion":1,"repository":"a/b","description":"d","maintainers":["u"}`, "invalid JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseDescriptor([]byte(test.data))
			if err == nil {
				t.Fatalf("ParseDescriptor accepted %s", test.name)
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), test.wantErr)
			}
		})
	}
}

func TestParseDescriptorBoundaries(t *testing.T) {
	t.Parallel()
	owner39 := strings.Repeat("o", 39)
	repo100 := strings.Repeat("r", 100)
	description200 := strings.Repeat("é", 200)
	maintainer39 := strings.Repeat("m", 39)
	topic35 := strings.Repeat("t", 35)
	maintainers10 := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8", "u9", "u10"}
	topics5 := []string{"t1", "t2", "t3", "t4", "t5"}

	valid := marshalDescriptor(t, map[string]any{
		"schemaVersion": 1,
		"repository":    owner39 + "/" + repo100,
		"description":   description200,
		"maintainers":   maintainers10,
		"topics":        topics5,
	})
	descriptor, err := ParseDescriptor(valid)
	if err != nil {
		t.Fatalf("ParseDescriptor rejected a boundary valid descriptor: %v", err)
	}
	if descriptor.Repository != owner39+"/"+repo100 {
		t.Errorf("repository = %q, want the 39 character owner and 100 character name", descriptor.Repository)
	}

	over := []struct {
		name   string
		fields map[string]any
	}{
		{"owner 40", map[string]any{"schemaVersion": 1, "repository": strings.Repeat("o", 40) + "/r", "description": "d", "maintainers": maintainers10}},
		{"repo 101", map[string]any{"schemaVersion": 1, "repository": "a/" + strings.Repeat("r", 101), "description": "d", "maintainers": maintainers10}},
		{"description 201", map[string]any{"schemaVersion": 1, "repository": "a/b", "description": strings.Repeat("é", 201), "maintainers": maintainers10}},
		{"maintainer 40", map[string]any{"schemaVersion": 1, "repository": "a/b", "description": "d", "maintainers": []string{strings.Repeat("m", 40)}}},
		{"maintainers 11", map[string]any{"schemaVersion": 1, "repository": "a/b", "description": "d", "maintainers": append(append([]string{}, maintainers10...), "u11")}},
		{"topic 36", map[string]any{"schemaVersion": 1, "repository": "a/b", "description": "d", "maintainers": maintainers10, "topics": []string{strings.Repeat("t", 36)}}},
		{"topics 6", map[string]any{"schemaVersion": 1, "repository": "a/b", "description": "d", "maintainers": maintainers10, "topics": append(append([]string{}, topics5...), "t6")}},
	}
	for _, test := range over {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseDescriptor(marshalDescriptor(t, test.fields)); err == nil {
				t.Fatalf("ParseDescriptor accepted %s", test.name)
			}
		})
	}

	boundaryValues := []struct {
		name  string
		value string
	}{
		{"maintainer 39", maintainer39},
		{"topic 35", topic35},
	}
	for _, test := range boundaryValues {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if !lowerSegmentPattern.MatchString(test.value) {
				t.Fatalf("value %q does not satisfy the identifier pattern", test.value)
			}
		})
	}
}

func TestParseDescriptorSizeLimit(t *testing.T) {
	t.Parallel()
	valid := "{" + validDescriptorBody + "}"
	atLimit := valid + strings.Repeat(" ", MaxDescriptorBytes-len(valid))
	if len(atLimit) != MaxDescriptorBytes {
		t.Fatalf("test setup: padded descriptor is %d bytes, want %d", len(atLimit), MaxDescriptorBytes)
	}
	descriptor, err := ParseDescriptor([]byte(atLimit))
	if err != nil {
		t.Fatalf("ParseDescriptor rejected a descriptor at the size limit: %v", err)
	}
	if descriptor.Repository != "a/b" {
		t.Errorf("repository = %q, want a/b", descriptor.Repository)
	}

	overLimit := valid + strings.Repeat(" ", MaxDescriptorBytes+1-len(valid))
	if _, err := ParseDescriptor([]byte(overLimit)); err == nil {
		t.Fatal("ParseDescriptor accepted a descriptor over the size limit")
	} else if !strings.Contains(err.Error(), "the limit is 16384 bytes") {
		t.Fatalf("error %q does not mention the size limit", err.Error())
	}
}

func TestParseDescriptorInvalidUTF8(t *testing.T) {
	t.Parallel()
	if _, err := ParseDescriptor([]byte("\xff{" + validDescriptorBody + "}")); err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("ParseDescriptor accepted invalid UTF-8, err = %v", err)
	}
	if _, err := ParseDescriptor([]byte(`{"schemaVersion":1,"repository":"a/` + "\xfe" + `","description":"d","maintainers":["u"]}`)); err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("ParseDescriptor accepted invalid UTF-8 inside a string, err = %v", err)
	}
}

func TestParseDescriptorRawControlByte(t *testing.T) {
	t.Parallel()
	if _, err := ParseDescriptor([]byte(`{"schemaVersion":1,"repository":"a/b","description":"a` + "\x7f" + `b","maintainers":["u"]}`)); err == nil || !strings.Contains(err.Error(), "control characters") {
		t.Fatalf("ParseDescriptor accepted a raw delete character, err = %v", err)
	}
}

func TestParseDescriptorRawFormatChar(t *testing.T) {
	t.Parallel()
	if _, err := ParseDescriptor([]byte(`{"schemaVersion":1,"repository":"a/b","description":"a` + "\u200bb" + `","maintainers":["u"]}`)); err == nil || !strings.Contains(err.Error(), "format characters") {
		t.Fatalf("ParseDescriptor accepted a raw zero width space, err = %v", err)
	}
}
