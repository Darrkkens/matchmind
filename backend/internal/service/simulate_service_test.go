package service

import (
	"context"
	"errors"
	"testing"

	"matchmind/internal/football"
)

type fakeSimProvider struct {
	football.FootballProvider
	in football.SimulationInput
}

func (f fakeSimProvider) SimulationInput(context.Context, string) (*football.SimulationInput, []string, error) {
	in := f.in
	return &in, []string{}, nil
}

type fakeAnalyst struct {
	adj   *football.SimulationAdjustment
	err   error
	calls int
}

func (f *fakeAnalyst) AnalyzeMatch(context.Context, string) (*football.SimulationAdjustment, error) {
	f.calls++
	return f.adj, f.err
}

func simProvider() fakeSimProvider {
	a, b := football.Team{ID: "a", Name: "Alpha"}, football.Team{ID: "b", Name: "Beta"}
	matches := []football.Match{{Date: "2026-09-01", HomeTeam: a, AwayTeam: b, HomeScore: 2, AwayScore: 1, Status: "finished"}}
	return fakeSimProvider{in: football.SimulationInput{Fixture: football.Fixture{Date: "2026-10-08", HomeTeam: a, AwayTeam: b}, ClubID: "a", Matches: matches, Standings: football.CalculateStandings([]football.Team{a, b}, matches)}}
}

func factor(sim *football.Simulation, key string) football.SimulationFactor {
	for _, f := range sim.Factors {
		if f.Key == key {
			return f
		}
	}
	return football.SimulationFactor{}
}

func TestSimulateWithAnalystAppliesAndCaches(t *testing.T) {
	analyst := &fakeAnalyst{adj: &football.SimulationAdjustment{Club: 1.1, Opponent: 0.95, Reason: "Alpha cria mais chances."}}
	s := &TeamService{Provider: simProvider(), Analyst: analyst}
	sim, err := s.Simulate(context.Background(), "a", 200, true)
	if err != nil {
		t.Fatal(err)
	}
	if f := factor(sim, "ai_analyst"); !f.Available || f.Club != 1.1 || analyst.calls != 1 {
		t.Fatalf("analyst factor %+v calls %d", f, analyst.calls)
	}
	fixture := sim.Fixture
	if cached := s.CachedSimulation("a", &fixture); cached != sim {
		t.Fatal("AI-reviewed simulation not cached for the chat")
	}
	other := fixture
	other.Date = "2026-10-20"
	if s.CachedSimulation("a", &other) != nil {
		t.Fatal("cache served a different fixture")
	}
}

func TestSimulateFallsBackWithoutAnalyst(t *testing.T) {
	analyst := &fakeAnalyst{err: errors.New("timeout")}
	s := &TeamService{Provider: simProvider(), Analyst: analyst}
	sim, err := s.Simulate(context.Background(), "a", 50, true)
	if err != nil {
		t.Fatal(err)
	}
	if f := factor(sim, "ai_analyst"); f.Available || f.Detail == "" {
		t.Fatalf("failed analyst must be reported, not used: %+v", f)
	}
	if s.CachedSimulation("a", &sim.Fixture) != nil {
		t.Fatal("a simulation without the AI step must not be cached as reviewed")
	}
	if _, err := s.Simulate(context.Background(), "a", 20000, true); !errors.Is(err, football.ErrSimulationRuns) || analyst.calls != 1 {
		t.Fatalf("invalid runs must fail before calling the AI: %v, calls %d", err, analyst.calls)
	}
	noAI, _ := s.Simulate(context.Background(), "a", 50, false)
	if f := factor(noAI, "ai_analyst"); f.Available || analyst.calls != 1 {
		t.Fatal("use_ai=false must not call the AI")
	}
}
