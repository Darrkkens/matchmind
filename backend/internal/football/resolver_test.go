package football

import (
	"strings"
	"testing"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		input, query, slug, id string
		invalid                bool
	}{
		{"Palmeiras", "Palmeiras", "", "", false},
		{"  Aurora FC  ", "Aurora FC", "", "", false},
		{"São Paulo", "São Paulo", "", "", false},
		{"https://www.sofascore.com/football/team/palmeiras/1963", "palmeiras", "palmeiras", "1963", false},
		{"https://example.org/football/team/harbor-united/10/?x=y#tab", "harbor united", "harbor-united", "10", false},
		{"http://127.0.0.1/football/team/aurora-fc/1", "aurora fc", "aurora-fc", "1", false},
		{"", "", "", "", true}, {"  ", "", "", "", true},
		{"https://example.org/team/palmeiras", "", "", "", true},
		{"https://example.org/football/team/palmeiras/abc", "", "", "", true},
		{"https://example.org/football/team/palmeiras/1/extra", "", "", "", true},
		{"https://example.org/football/team/a%2Fb/1", "", "", "", true},
		{"https://example.org/football/team/%zz/1", "", "", "", true},
		{"https://user:secret@example.org/football/team/a/1", "", "", "", true},
		{"https:///football/team/a/1", "", "", "", true},
		{"file:///etc/passwd", "", "", "", true},
		{"javascript:alert(1)", "", "", "", true},
		{"//localhost/football/team/a/1", "", "", "", true},
		{"Pal\x00meiras", "", "", "", true},
		{strings.Repeat("x", 2049), "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseReference(tt.input)
			if (err != nil) != tt.invalid {
				t.Fatalf("error=%v", err)
			}
			if !tt.invalid && (got.Query != tt.query || got.Slug != tt.slug || got.ExternalID != tt.id) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
