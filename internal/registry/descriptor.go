package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const SchemaVersion = 1

const (
	MaxDescriptorBytes = 16384
	MaxDescriptors     = 10000
	MaxOwnerLength     = 39
	MaxRepoLength      = 100
	MaxDescriptionLen  = 200
	MaxMaintainers     = 10
	MaxMaintainerLen   = 39
	MaxTopics          = 5
	MaxTopicLen        = 35
)

var (
	ownerPattern          = regexp.MustCompile(`^[0-9A-Za-z]+(-[0-9A-Za-z]+)*$`)
	repositoryNamePattern = regexp.MustCompile(`^[0-9A-Za-z_.-]+$`)
	lowerSegmentPattern   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

type Descriptor struct {
	SchemaVersion int
	Repository    string
	Description   string
	Maintainers   []string
	Topics        []string
}

func ParseDescriptor(data []byte) (Descriptor, error) {
	var descriptor Descriptor
	if len(data) > MaxDescriptorBytes {
		return descriptor, fmt.Errorf("descriptor is %d bytes, the limit is %d bytes", len(data), MaxDescriptorBytes)
	}
	if !utf8.Valid(data) {
		return descriptor, errors.New("descriptor is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	seen := map[string]bool{}
	if err := expectDelim(decoder, '{'); err != nil {
		return descriptor, err
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return descriptor, invalidJSON(err)
		}
		key, ok := token.(string)
		if !ok {
			return descriptor, errors.New("invalid JSON: object keys must be strings")
		}
		if seen[key] {
			return descriptor, fmt.Errorf("duplicate field %q", key)
		}
		seen[key] = true
		switch key {
		case "schemaVersion":
			descriptor.SchemaVersion, err = decodeSchemaVersion(decoder)
		case "repository":
			descriptor.Repository, err = decodeString(decoder, key)
		case "description":
			descriptor.Description, err = decodeString(decoder, key)
		case "maintainers":
			descriptor.Maintainers, err = decodeStringArray(decoder, key)
		case "topics":
			descriptor.Topics, err = decodeStringArray(decoder, key)
		default:
			err = fmt.Errorf("unknown field %q", key)
		}
		if err != nil {
			return descriptor, err
		}
	}
	if err := expectDelim(decoder, '}'); err != nil {
		return descriptor, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return descriptor, errors.New("trailing data after the JSON object")
		}
		return descriptor, invalidJSON(err)
	}
	if err := descriptor.validate(seen); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func (d Descriptor) validate(seen map[string]bool) error {
	for _, field := range []string{"schemaVersion", "repository", "description", "maintainers"} {
		if !seen[field] {
			return fmt.Errorf("missing required field %q", field)
		}
	}
	if _, _, err := SplitRepository(d.Repository); err != nil {
		return err
	}
	if err := ValidateDescription(d.Description); err != nil {
		return err
	}
	if err := validateIdentifiers("maintainers", d.Maintainers, 1, MaxMaintainers, MaxMaintainerLen); err != nil {
		return err
	}
	return validateIdentifiers("topics", d.Topics, 0, MaxTopics, MaxTopicLen)
}

func SplitRepository(repository string) (string, string, error) {
	owner, name, found := strings.Cut(repository, "/")
	if !found || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("repository %q must be owner/repo without scheme, host, version, query, or fragment", repository)
	}
	if len(owner) > MaxOwnerLength {
		return "", "", fmt.Errorf("repository owner %q is longer than %d characters", owner, MaxOwnerLength)
	}
	if !ownerPattern.MatchString(owner) {
		return "", "", fmt.Errorf("repository owner %q must be alphanumeric segments separated by single hyphens", owner)
	}
	if len(name) > MaxRepoLength {
		return "", "", fmt.Errorf("repository name %q is longer than %d characters", name, MaxRepoLength)
	}
	if name == "." || name == ".." {
		return "", "", fmt.Errorf("repository name %q is not allowed", name)
	}
	if !repositoryNamePattern.MatchString(name) {
		return "", "", fmt.Errorf("repository name %q must only contain ASCII letters, digits, underscores, hyphens, and dots", name)
	}
	return owner, name, nil
}

func ValidateDescription(description string) error {
	if description == "" {
		return errors.New(`field "description" is required`)
	}
	if strings.TrimSpace(description) != description {
		return errors.New(`field "description" must not have leading or trailing whitespace`)
	}
	if count := utf8.RuneCountInString(description); count > MaxDescriptionLen {
		return fmt.Errorf("field %q is %d characters, the limit is %d", "description", count, MaxDescriptionLen)
	}
	for _, char := range description {
		if unicode.IsControl(char) {
			return fmt.Errorf("field %q must not contain control characters, found %q", "description", char)
		}
		if unicode.Is(unicode.Cf, char) {
			return fmt.Errorf("field %q must not contain format characters, found %q", "description", char)
		}
		if char == '\u2028' || char == '\u2029' {
			return fmt.Errorf("field %q must be a single line, found a line separator character", "description")
		}
	}
	return nil
}

func validateIdentifiers(field string, values []string, minCount, maxCount, maxLen int) error {
	if len(values) < minCount || len(values) > maxCount {
		return fmt.Errorf("field %q requires between %d and %d entries, got %d", field, minCount, maxCount, len(values))
	}
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if seen[value] {
			return fmt.Errorf("field %q has duplicate entry %q", field, value)
		}
		seen[value] = true
		if len(value) > maxLen {
			return fmt.Errorf("field %q entry %q is longer than %d characters", field, value, maxLen)
		}
		if !lowerSegmentPattern.MatchString(value) {
			return fmt.Errorf("field %q entry %q must be lowercase letters and digits separated by single hyphens", field, value)
		}
	}
	return nil
}

func expectDelim(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return invalidJSON(err)
	}
	if token != want {
		return fmt.Errorf("invalid JSON: expected %q, got %v", want, token)
	}
	return nil
}

func decodeSchemaVersion(decoder *json.Decoder) (int, error) {
	token, err := decoder.Token()
	if err != nil {
		return 0, invalidJSON(err)
	}
	if token == nil {
		return 0, errors.New(`field "schemaVersion" must not be null`)
	}
	number, ok := token.(json.Number)
	if !ok {
		return 0, fmt.Errorf("field %q must be a number, got %v", "schemaVersion", token)
	}
	value, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("field %q must be the whole number %d", "schemaVersion", SchemaVersion)
	}
	if value != SchemaVersion {
		return 0, fmt.Errorf("unsupported schemaVersion %d, only %d is supported", value, SchemaVersion)
	}
	return int(value), nil
}

func decodeString(decoder *json.Decoder, field string) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", invalidJSON(err)
	}
	if token == nil {
		return "", fmt.Errorf("field %q must not be null", field)
	}
	value, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string, got %v", field, token)
	}
	return value, nil
}

func decodeStringArray(decoder *json.Decoder, field string) ([]string, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, invalidJSON(err)
	}
	if token == nil {
		return nil, fmt.Errorf("field %q must not be null", field)
	}
	if token != json.Delim('[') {
		return nil, fmt.Errorf("field %q must be an array, got %v", field, token)
	}
	var values []string
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, invalidJSON(err)
		}
		value, ok := token.(string)
		if !ok {
			if token == nil {
				return nil, fmt.Errorf("field %q entries must not be null", field)
			}
			return nil, fmt.Errorf("field %q entries must be strings, got %v", field, token)
		}
		values = append(values, value)
	}
	if err := expectDelim(decoder, ']'); err != nil {
		return nil, err
	}
	return values, nil
}

func invalidJSON(err error) error {
	return fmt.Errorf("invalid JSON: %w", err)
}
