package ai

import (
	"context"
	"encoding/json"
	"errors"
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
	facts := `{"club":"Palmeiras","opponent":"Bahia","ga":0.75}`
	adj, err := analystServer(t, `{"reason":"O Palmeiras sofre 0,75 gol por jogo.","favored":"Palmeiras","strength":"moderado"}`).AnalyzeMatch(context.Background(), facts)
	if err != nil || adj.Club != 1.06 || adj.Opponent != 0.94 || adj.Model != "gemma3:4b" {
		t.Fatalf("%+v %v", adj, err)
	}
	// Favoring the opponent flips the edge; "nenhum" leaves both sides unchanged.
	adj, _ = analystServer(t, `{"reason":"O Bahia chega melhor.","favored":"Bahia","strength":"leve"}`).AnalyzeMatch(context.Background(), facts)
	if adj.Club != 0.97 || adj.Opponent != 1.03 {
		t.Fatalf("opponent edge %+v", adj)
	}
	for _, bad := range []string{`{"reason":"x","favored":"Santos","strength":"leve"}`, `{"reason":"  ","favored":"nenhum","strength":"leve"}`, `{"reason":"x","favored":"nenhum","strength":"enorme"}`, `not json`} {
		if _, err := analystServer(t, bad).AnalyzeMatch(context.Background(), facts); err != ErrInvalidResponse {
			t.Fatalf("%s accepted", bad)
		}
	}
	// The reason must name the club it favors.
	if _, err := analystServer(t, `{"reason":"O Bahia sofre muitos gols.","favored":"Palmeiras","strength":"leve"}`).AnalyzeMatch(context.Background(), facts); !errors.Is(err, ErrUngroundedReason) {
		t.Fatalf("reason about the other club accepted: %v", err)
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

func TestBuildAnalystFactsLabelsVenueRoles(t *testing.T) {
	home, away := football.Team{ID: "h", Name: "Remo"}, football.Team{ID: "a", Name: "Grêmio"}
	prelim := &football.Simulation{ClubID: "a", Fixture: football.Fixture{Date: "2026-10-08", HomeTeam: home, AwayTeam: away}}
	club := &football.SeasonStats{Home: &football.SplitRecord{Played: 14, Wins: 9}, Away: &football.SplitRecord{Played: 14, Losses: 10}}
	opp := &football.SeasonStats{Home: &football.SplitRecord{Played: 14, Wins: 3}, Away: &football.SplitRecord{Played: 14, Wins: 1}}
	facts, _ := BuildAnalystFacts(prelim, club, opp)
	if !strings.Contains(facts, `"match":"Remo (em casa) x Grêmio (fora)`) || !strings.Contains(facts, "Grêmio fora: 14 jogos, 0 vitórias, 0 empates, 10 derrotas") || !strings.Contains(facts, "Remo em casa: 14 jogos, 3 vitórias") || strings.Contains(facts, "Grêmio em casa") {
		t.Fatalf("facts %s", facts)
	}
}

func TestAnalystReasonMustCiteOnlyFactNumbers(t *testing.T) {
	facts := `{"x":"Remo em casa: 14 jogos, 18 gols marcados","edge":"+14%","ga":0.75}`
	if !numbersGrounded("O Remo marcou 18 gols em 14 jogos em casa; defesa 0,75 e mando 14%.", facts) {
		t.Fatal("grounded reason rejected")
	}
	if numbersGrounded("O Remo tem média de 1.8 gols em casa.", facts) {
		t.Fatal("invented number accepted")
	}
	if adj, err := analystServer(t, `{"reason":"O Remo marca 1.8 gols por jogo.","favored":"nenhum","strength":"leve"}`).AnalyzeMatch(context.Background(), `{"club":"Remo","opponent":"Grêmio","x":"14 jogos, 18 gols"}`); !errors.Is(err, ErrUngroundedReason) {
		t.Fatalf("ungrounded adjustment applied: %+v %v", adj, err)
	}
}

func TestAnalystReasonDirectionMustMatchMultipliers(t *testing.T) {
	facts := `{"club":"CR Flamengo","opponent":"Santos FC"}`
	if directionConsistent("Um pequeno ajuste favorece o Santos FC.", facts, 1.02, 0.94) {
		t.Fatal("reason favoring the opponent accepted while numbers favor the club")
	}
	if !directionConsistent("O ajuste favorece o CR Flamengo.", facts, 1.02, 0.94) || !directionConsistent("Defesas equilibradas.", facts, 1.02, 0.94) {
		t.Fatal("consistent reason rejected")
	}
}
