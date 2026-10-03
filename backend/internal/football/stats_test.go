package football

import (
	"reflect"
	"testing"
)

func TestFormAwayLossAndIgnoredMatches(t *testing.T) {
	a, b, c := Team{ID: "a"}, Team{ID: "b"}, Team{ID: "c"}
	m := []Match{{HomeTeam: b, AwayTeam: a, HomeScore: 3, AwayScore: 1, Status: "finished"}, {HomeTeam: a, AwayTeam: b, HomeScore: 0, AwayScore: 0, Status: "scheduled"}, {HomeTeam: b, AwayTeam: c, HomeScore: 9, AwayScore: 0, Status: "finished"}}
	f := CalculateForm("a", m)
	if f.Played != 1 || f.Losses != 1 || f.GoalsScored != 1 || f.GoalsConceded != 3 || f.PointsPercentage != 0 {
		t.Fatalf("unexpected %+v", f)
	}
	f = CalculateForm("a", nil)
	if f.Played != 0 || f.AverageGoals != 0 || f.PointsPercentage != 0 || f.Sequence == nil {
		t.Fatalf("invalid empty form %+v", f)
	}
}

func TestStandingsOrderAndTieBreakers(t *testing.T) {
	a, b, c, d := Team{ID: "a", Name: "Alfa"}, Team{ID: "b", Name: "Beta"}, Team{ID: "c", Name: "Gama"}, Team{ID: "d", Name: "Delta"}
	m := []Match{
		{HomeTeam: a, AwayTeam: b, HomeScore: 3, AwayScore: 0, Status: "finished"},
		{HomeTeam: c, AwayTeam: d, HomeScore: 1, AwayScore: 0, Status: "finished"},
		{HomeTeam: b, AwayTeam: c, HomeScore: 1, AwayScore: 1, Status: "finished"},
		{HomeTeam: d, AwayTeam: a, HomeScore: 9, AwayScore: 9, Status: "scheduled"},
		{HomeTeam: a, AwayTeam: Team{ID: "x"}, HomeScore: 5, AwayScore: 0, Status: "finished"},
	}
	table := CalculateStandings([]Team{d, c, b, a}, m)
	got := []string{}
	for _, row := range table {
		got = append(got, row.TeamID)
	}
	if !reflect.DeepEqual(got, []string{"c", "a", "b", "d"}) {
		t.Fatalf("order %v", got)
	}
	if table[1].Position != 2 || table[1].Points != 3 || table[1].GoalDifference != 3 || table[1].Played != 1 || table[3].Losses != 1 || table[3].Played != 1 {
		t.Fatalf("rows %+v", table)
	}
	// Equal points and wins: goal difference, then goals scored, decide.
	e, f := Team{ID: "e", Name: "E"}, Team{ID: "f", Name: "F"}
	table = CalculateStandings([]Team{e, f, a, b}, []Match{
		{HomeTeam: e, AwayTeam: a, HomeScore: 1, AwayScore: 0, Status: "finished"},
		{HomeTeam: f, AwayTeam: b, HomeScore: 3, AwayScore: 2, Status: "finished"},
	})
	if table[0].TeamID != "f" || table[1].TeamID != "e" {
		t.Fatalf("goals-for tie-break %+v", table)
	}
	if empty := CalculateStandings(nil, nil); empty == nil || len(empty) != 0 {
		t.Fatal("empty table must be a non-nil empty slice")
	}
}
