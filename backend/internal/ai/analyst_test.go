package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"matchmind/internal/football"
)

func analystServer(t *testing.T, content string) *Ollama {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "FACTS") || !strings.Contains(string(body), "match analyst") {
			t.Errorf("unexpected request %s", body)
		}
		resp, _ := json.Marshal(map[string]any{"done": true, "message": map[string]string{"content": content}})
		_, _ = w.Write(resp)
	}))
	t.Cleanup(srv.Close)
	o, err := NewOllama(srv.URL, "gemma3:4b", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestAnalyzeMatch(t *testing.T) {
	adj, err := analystServer(t, `{"club":1.06,"opponent":0.97,"reason":"O Palmeiras sofre 0,75 gol por jogo."}`).AnalyzeMatch(context.Background(), `{"club":"A"}`)
	if err != nil || adj.Club != 1.06 || adj.Opponent != 0.97 || adj.Model != "gemma3:4b" {
		t.Fatalf("%+v %v", adj, err)
	}
	for _, bad := range []string{`{"club":1.0,"reason":"x"}`, `{"club":1,"opponent":1,"reason":"  "}`, `not json`} {
		if _, err := analystServer(t, bad).AnalyzeMatch(context.Background(), `{}`); err != ErrInvalidResponse {
			t.Fatalf("%s accepted", bad)
		}
	}
}

func TestBuildAnalystFactsSkipsRandomnessAndUnusedFactors(t *testing.T) {
	a, b := football.Team{ID: "a", Name: "Alpha"}, football.Team{ID: "b", Name: "Beta"}
	prelim := &football.Simulation{ClubID: "b", Fixture: football.Fixture{Date: "2026-10-08", HomeTeam: a, AwayTeam: b}, ExpectedClub: 1.1, ExpectedOpp: 1.4,
		Factors: []football.SimulationFactor{{Key: "form", Club: 1.05, Opponent: 1, Available: true, Detail: "forma"}, {Key: "injuries", Club: 1, Opponent: 1}, {Key: "randomness", Available: true}}}
	facts, err := BuildAnalystFacts(prelim, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(facts, `"club":"Beta"`) || !strings.Contains(facts, `"club_plays_at_home":false`) || !strings.Contains(facts, `"form"`) || strings.Contains(facts, "injuries") || strings.Contains(facts, "randomness") {
		t.Fatalf("facts %s", facts)
	}
}

func TestSimulationContextKeepsSidesWhenClubIsAway(t *testing.T) {
	home, away := football.Team{ID: "h", Name: "Home FC"}, football.Team{ID: "a", Name: "Away FC"}
	sim := &football.Simulation{Runs: 50, ClubID: "a", Fixture: football.Fixture{Date: "2026-10-08", HomeTeam: home, AwayTeam: away}, Win: 40, Draw: 30, Loss: 30,
		Scorelines: []football.Scoreline{{Club: 2, Opponent: 1, Percent: 12.5}},
		Factors:    []football.SimulationFactor{{Key: "form", Club: 1.02, Opponent: 1, Available: true, Detail: "f"}, {Key: "venue", Club: 0.9, Opponent: 1.1, Available: true, Detail: "v"}, {Key: "injuries", Club: 1, Opponent: 1, Detail: "sem fonte"}}}
	ctx, err := BuildContextWithSimulation(&football.Snapshot{Team: &away}, "explique a simulação", sim)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Away FC vence em 40,0%", "Home FC vence em 30,0%", "Home FC 1 x 2 Away FC: 12,5%", "Lesionados: sem fonte"} {
		if !strings.Contains(ctx, want) {
			t.Fatalf("missing %q in %s", want, ctx)
		}
	}
	// The venue factor moved more than form, so it is listed first.
	if strings.Index(ctx, "Campanha em casa e fora") > strings.Index(ctx, "Últimos 5 jogos") {
		t.Fatalf("factors not ordered by weight: %s", ctx)
	}
}

func TestSimulationSummaryUsesFixtureOrder(t *testing.T) {
	home, away := football.Team{ID: "h", Name: "Home FC"}, football.Team{ID: "a", Name: "Away FC"}
	sim := &football.Simulation{Runs: 10000, ClubID: "a", Fixture: football.Fixture{HomeTeam: home, AwayTeam: away}, Win: 40, Draw: 30, Loss: 30,
		Scorelines: []football.Scoreline{{Club: 2, Opponent: 1, Percent: 12.5}}}
	got := SimulationSummary(sim)
	for _, want := range []string{"10.000 jogos simulados", "Away FC 40,0% · empate 30,0% · Home FC 30,0%", "(Home FC x Away FC): 1 x 2 (12,5%)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
