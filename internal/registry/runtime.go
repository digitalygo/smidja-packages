package registry

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var packageIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,127}$`)
var releaseTargets = map[string]bool{"linux/amd64": true, "linux/arm64": true, "darwin/amd64": true, "darwin/arm64": true}

func decodeReleaseFeeds(decoder *json.Decoder) (map[string]string, error) {
	if err := expectDelim(decoder, '{'); err != nil {
		return nil, err
	}
	feeds := map[string]string{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, invalidJSON(err)
		}
		target, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("release feed target must be a string")
		}
		if _, exists := feeds[target]; exists {
			return nil, fmt.Errorf("duplicate release feed target %q", target)
		}
		value, err := decodeString(decoder, "releaseFeeds")
		if err != nil {
			return nil, err
		}
		feeds[target] = value
	}
	if err := expectDelim(decoder, '}'); err != nil {
		return nil, err
	}
	return feeds, nil
}

func (d Descriptor) validateRuntime(seen map[string]bool) error {
	for _, field := range []string{"id", "scopes", "releaseFeeds"} {
		if !seen[field] {
			return fmt.Errorf("missing required field %q", field)
		}
	}
	if !packageIDPattern.MatchString(d.ID) {
		return fmt.Errorf("invalid package id")
	}
	if len(d.Scopes) == 0 || len(d.Scopes) > 2 {
		return fmt.Errorf("scopes must contain manager, client or both")
	}
	scopes := map[string]bool{}
	for _, scope := range d.Scopes {
		if (scope != "manager" && scope != "client") || scopes[scope] {
			return fmt.Errorf("invalid or duplicate scope %q", scope)
		}
		scopes[scope] = true
	}
	if len(d.ReleaseFeeds) == 0 || len(d.ReleaseFeeds) > 4 {
		return fmt.Errorf("releaseFeeds requires one to four targets")
	}
	for target, feed := range d.ReleaseFeeds {
		if !releaseTargets[target] {
			return fmt.Errorf("unsupported release target %q", target)
		}
		parsed, err := url.Parse(feed)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(feed, "#") || strings.ContainsAny(feed, " <>`\"\\") {
			return fmt.Errorf("release feed must be an HTTPS URL without credentials, query or fragment")
		}
	}
	return nil
}
