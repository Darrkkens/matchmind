package football

import (
	"errors"
	"math"
	"testing"
)

// Invented league: Strong scores a lot and concedes little, Weak the opposite.
func simInput(t *testing.T) SimulationInput {
	t.Helper()
	strong, weak, mid := Team{ID: "s", Name: "Strong"}, Team{ID: "w", Name: "Weak"}, Team{ID: "m", Name: "Mid"}
	matches := []Match{
		{Date: "2026-09-28", HomeTeam: strong, AwayTeam: mid, HomeScore: 3, AwayScore: 0, Status: "finished"},
		{Date: "2026-09-27", HomeTeam: mid, AwayTeam: weak, HomeScore: 2, AwayScore: 0, Status: "finished"},
		{Date: "2026-09-20", HomeTeam: weak, AwayTeam: strong, HomeScore: 0, AwayScore: 2, Status: "finished"},
		{Date: "2026-09-13", HomeTeam: strong, AwayTeam: mid, HomeScore: 2, AwayScore: 1, Status: "finished"},
		{Date: "2026-09-06", HomeTeam: mid, AwayTeam: weak, HomeScore: 1, AwayScore: 1, Status: "finished"},
	}
	return SimulationInput{
		Fixture:   Fixture{Date: "2026-09-30", HomeTeam: strong, AwayTeam: weak},
		ClubID:    "s",
		Matches:   matches,
		Standings: CalculateStandings([]Team{strong, weak, mid}, matches),
	}
}

func TestSimulateIsDeterministicAndFavorsTheStrongerSide(t *testing.T) {
	in := simInput(t)
	a, err := Simulate(in, 2000)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Simulate(in, 2000)
	if a.Win != b.Win || a.Draw != b.Draw || a.Seed != b.Seed {
		t.Fatal("same input must give the same result")
	}
	if math.Abs(a.Win+a.Draw+a.Loss-100) > 0.11 || a.Win <= a.Loss || a.ExpectedClub <= a.ExpectedOpp {
		t.Fatalf("result %+v", a)
	}
	if len(a.Scorelines) == 0 || len(a.Scorelines) > 5 || len(a.ClubRecent) != 3 || a.ClubRecent[0] != "W" {
		t.Fatalf("scorelines %+v recent %v", a.Scorelines, a.ClubRecent)
	}
	keys := map[string]SimulationFactor{}
	for _, f := range a.Factors {
		keys[f.Key] = f
	}
	// The league home edge is its own factor and favors the home side (Strong plays at home).
	if h := keys["home_edge"]; !h.Available || h.Club <= 1 || h.Opponent >= 1 {
		t.Fatalf("home edge %+v", h)
	}
	if s := keys["season"]; s.ClubValue <= 0 || s.OpponentValue <= 0 {
		t.Fatalf("base expected goals missing %+v", s)
	}
	for _, k := range []string{"season", "home_edge", "form", "rest", "injuries", "head_to_head", "randomness"} {
		if _, ok := keys[k]; !ok {
			t.Fatalf("missing factor %s", k)
		}
	}
	// Two days of rest for Strong (played on the 28th) must cost it, Weak rested 3 days.
	if r := keys["rest"]; !r.Available || r.Club >= 1 {
		t.Fatalf("rest %+v", r)
	}
	if keys["injuries"].Available {
		t.Fatal("injuries have no source and must not claim to be used")
	}
	// The season meeting (Weak 0–2 Strong) counts as head-to-head.
	if h := keys["head_to_head"]; !h.Available || h.Club <= 1 {
		t.Fatalf("h2h %+v", h)
	}
}

func TestSimulateHeadToHeadHistoryAndValidation(t *testing.T) {
	in := simInput(t)
	base, _ := Simulate(in, 1000)
	in.History = &HeadToHead{Record: Record{Played: 30, Wins: 2, Draws: 3, Losses: 25}}
	worse, _ := Simulate(in, 1000)
	if worse.ExpectedClub >= base.ExpectedClub {
		t.Fatalf("a losing history must lower the club's goals: %v vs %v", worse.ExpectedClub, base.ExpectedClub)
	}
	for _, runs := range []int{0, -1, 10001} {
		if _, err := Simulate(in, runs); !errors.Is(err, ErrSimulationRuns) {
			t.Fatalf("runs %d accepted", runs)
		}
	}
	in.ClubID = "nobody"
	if _, err := Simulate(in, 50); !errors.Is(err, ErrNotFound) {
		t.Fatal("club outside the fixture accepted")
	}
	few, _ := Simulate(simInput(t), 50)
	if len(few.Notes) == 0 {
		t.Fatal("small run counts must warn about noise")
	}
}

func TestSimulateUsesSeasonDetailVenueAndAnalyst(t *testing.T) {
	in := simInput(t)
	season := func(goals, sot, ga, sota float64, home, away SplitRecord) *SeasonStats {
		m := func(key string, v, league float64) SeasonMetric {
			return SeasonMetric{Key: key, Value: v, League: league, Rank: 1, Clubs: 20}
		}
		return &SeasonStats{Source: "test", AsOf: "2026-09-20", Matches: 10, Home: &home, Away: &away,
			Metrics: []SeasonMetric{m("goals", goals, 1.3), m("shots_on_target", sot, 4.5), m("goals_against", ga, 1.3), m("shots_on_target_against", sota, 4.5)}}
	}
	in.ClubSeason = season(2.0, 6.0, 0.7, 3.5, SplitRecord{Played: 5, Wins: 5, GoalsFor: 14, GoalsAgainst: 2}, SplitRecord{Played: 5, Wins: 2, GoalsFor: 6, GoalsAgainst: 5})
	in.OpponentSeason = season(0.9, 3.5, 1.8, 5.5, SplitRecord{Played: 5, Wins: 1, GoalsFor: 6, GoalsAgainst: 9}, SplitRecord{Played: 5, Losses: 4, GoalsFor: 3, GoalsAgainst: 9})
	in.Analyst = &SimulationAdjustment{Club: 1.5, Opponent: 0.5, Reason: "teste"}
	sim, err := Simulate(in, 500)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]SimulationFactor{}
	for _, f := range sim.Factors {
		got[f.Key] = f
	}
	if f := got["season_detail"]; !f.Available || f.Club < 0.85 || f.Club > 1.15 {
		t.Fatalf("season detail %+v", f)
	}
	if f := got["venue"]; !f.Available {
		t.Fatalf("venue %+v", f)
	}
	// The analyst's adjustment is clamped to ±6% whatever it asks for.
	if f := got["ai_analyst"]; !f.Available || f.Club != 1.06 || f.Opponent != 0.94 || f.Detail != "teste" {
		t.Fatalf("analyst %+v", f)
	}
}
