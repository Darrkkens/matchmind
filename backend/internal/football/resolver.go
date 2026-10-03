package football

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Reference struct {
	Query      string
	Slug       string
	ExternalID string
}

var teamPath = regexp.MustCompile(`^/football/team/([\p{L}\p{N}]+(?:-[\p{L}\p{N}]+)*)/([0-9]+)/?$`)

// ParseReference never performs network I/O. Even localhost URLs are only text.
func ParseReference(input string) (Reference, error) {
	input = strings.TrimSpace(input)
	if input == "" || len(input) > 2048 || !utf8.ValidString(input) || strings.ContainsFunc(input, unicode.IsControl) {
		return Reference{}, ErrInvalidInput
	}
	if strings.Contains(input, ":") || strings.Contains(input, "/") {
		u, err := url.Parse(input)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
			return Reference{}, ErrInvalidInput
		}
		parts := teamPath.FindStringSubmatch(u.Path)
		if len(parts) != 3 {
			return Reference{}, ErrInvalidInput
		}
		return Reference{Query: strings.ReplaceAll(parts[1], "-", " "), Slug: parts[1], ExternalID: parts[2]}, nil
	}
	if utf8.RuneCountInString(input) > 100 || strings.ContainsAny(input, "?#\\") {
		return Reference{}, ErrInvalidInput
	}
	return Reference{Query: input}, nil
}
