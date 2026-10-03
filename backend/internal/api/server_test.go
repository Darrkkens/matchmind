package api

import (
	"context"
	"encoding/json"
	"matchmind/internal/ai"
	"matchmind/internal/football"
	"matchmind/internal/footballtest"
	"matchmind/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubAI struct{ offline bool }

func (s stubAI) Chat(context.Context, string, string) (ai.Answer, error) {
	if s.offline {
		return ai.Answer{}, ai.ErrUnavailable
	}
	return ai.Answer{Answer: "FATO: 5 gols.", SourcesUsed: []string{"recent_form"}}, nil
}
func (s stubAI) Healthy(context.Context) bool { return !s.offline }
func handler(offline bool) http.Handler {
	teams := &service.TeamService{Provider: footballtest.NewProvider(), Source: "demo"}
	return New(teams, &service.ChatService{Teams: teams, AI: stubAI{offline}}, []string{"http://localhost:5173"})
}
func call(h http.Handler, method, path, body, origin, contentType string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAPIRoutesAndValidation(t *testing.T) {
	tests := []struct {
		name, method, path, body, contentType string
		status                                int
	}{
		{"health", "GET", "/api/health", "", "", 200},
		{"resolve", "POST", "/api/team/resolve", `{"input":"Palmeiras"}`, "application/json", 200},
		{"chat", "POST", "/api/chat", `{"team_id":"demo-palmeiras","question":"Goals?"}`, "application/json", 200},
		{"context forbidden", "POST", "/api/chat", `{"team_id":"demo-palmeiras","question":"Goals?","context":"evil"}`, "application/json", 400},
		{"invalid", "POST", "/api/team/resolve", `{"input":"file:///etc/passwd"}`, "application/json", 400},
		{"unknown", "POST", "/api/team/resolve", `{"input":"Nobody"}`, "application/json", 404},
		{"empty question", "POST", "/api/chat", `{"team_id":"demo-palmeiras","question":" "}`, "application/json", 400},
		{"invalid json", "POST", "/api/chat", `{`, "application/json", 400},
		{"trailing json", "POST", "/api/team/resolve", `{"input":"Palmeiras"} {}`, "application/json", 400},
		{"null", "POST", "/api/team/resolve", `null`, "application/json", 400},
		{"media type", "POST", "/api/team/resolve", `{}`, "text/plain", 415},
		{"too large", "POST", "/api/team/resolve", `{"input":"` + strings.Repeat("a", 17000) + `"}`, "application/json", 413},
		{"method", "GET", "/api/chat", "", "", 405},
		{"provider field removed", "POST", "/api/team/resolve", `{"input":"Palmeiras","provider":"demo"}`, "application/json", 400},
		{"standings unsupported", "GET", "/api/standings", "", "", 502},
		{"lineups unsupported", "GET", "/api/lineups/123", "", "", 404},
		{"lineups invalid ref", "GET", "/api/lineups/abc", "", "", 400},
		{"lineups path traversal", "GET", "/api/lineups/..%2F..%2Fetc", "", "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := call(handler(false), tt.method, tt.path, tt.body, "", tt.contentType)
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
func TestCORSAndRateLimit(t *testing.T) {
	h := handler(false)
	w := call(h, "OPTIONS", "/api/chat", "", "http://localhost:5173", "")
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatal("preflight failed")
	}
	w = call(h, "POST", "/api/chat", `{}`, "https://evil.example", "application/json")
	if w.Code != 403 {
		t.Fatal("foreign origin accepted")
	}
	for i := 0; i < 120; i++ {
		w = call(h, "GET", "/api/health", "", "", "")
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	w = call(h, "GET", "/api/health", "", "", "")
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("rate limit missing")
	}
}
func TestOfflineStillServesClub(t *testing.T) {
	h := handler(true)
	w := call(h, "GET", "/api/health", "", "", "")
	var health map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &health)
	if health["status"] != "degraded" || health["ollama"] != false || health["football_provider"] != true {
		t.Fatalf("%v", health)
	}
	w = call(h, "POST", "/api/team/resolve", `{"input":"Palmeiras"}`, "", "application/json")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = call(h, "POST", "/api/chat", `{"team_id":"demo-palmeiras","question":"Goals?"}`, "", "application/json")
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Verifique se o Ollama está rodando") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestRemoteFailureIsExplicit(t *testing.T) {
	remote := &service.TeamService{Provider: unavailableProvider{footballtest.NewProvider()}, Source: "openfootball"}
	h := New(remote, &service.ChatService{Teams: remote, AI: stubAI{}}, nil)
	for _, req := range [][3]string{{"POST", "/api/team/resolve", `{"input":"Palmeiras"}`}, {"POST", "/api/chat", `{"team_id":"demo-palmeiras","question":"Goals?"}`}} {
		w := call(h, req[0], req[1], req[2], "", "application/json")
		if w.Code != 503 || w.Header().Get("Retry-After") == "" {
			t.Fatalf("%s: expected explicit remote failure, got %d: %s", req[1], w.Code, w.Body.String())
		}
	}
}

type unavailableProvider struct{ football.FootballProvider }

func (p unavailableProvider) SearchTeam(context.Context, string) ([]football.Team, error) {
	return nil, football.ErrProviderUnavailable
}
func (p unavailableProvider) GetTeam(context.Context, string) (*football.Team, error) {
	return nil, football.ErrProviderUnavailable
}
