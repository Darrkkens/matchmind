package football

import "testing"

func TestNextMatchAvailabilityCardRules(t *testing.T) {
	club, rival := Team{ID: "c", Name: "Club"}, Team{ID: "r", Name: "Rival"}
	season := &SeasonStats{AsOf: "2026-09-20", Bookings: []PlayerBookings{
		{Player: "Gustavo Gómez", Yellow: 6, MinutesPct: 90, OnOff: 0.8}, // 6th yellow in the last match
		{Player: "Andreas Pereira", Yellow: 4},                           // yellow, not a multiple of three
		{Player: "Raphael Veiga", Yellow: 2},                             // one away
		{Player: "Flaco López", Yellow: 1, Red: 1},                       // sent off
	}}
	last := &Match{Date: "2026-09-20", HomeTeam: rival, AwayTeam: club, Cards: []Card{
		{Side: "away", Player: "G. Gómez", Color: "yellow"},
		{Side: "away", Player: "Andreas Pereira", Color: "yellow"},
		{Side: "away", Player: "Flaco López", Color: "red"},
		{Side: "home", Player: "Someone Else", Color: "red"}, // the other side
	}}
	a := NextMatchAvailability("c", last, season)
	if len(a.Suspended) != 2 || a.Suspended[0].Player != "Gustavo Gómez" || a.Suspended[0].MinutesPct != 90 || a.Suspended[1].Player != "Flaco López" {
		t.Fatalf("suspended %+v", a.Suspended)
	}
	if len(a.AtRisk) != 1 || a.AtRisk[0] != "Raphael Veiga" {
		t.Fatalf("at risk %v", a.AtRisk)
	}
	// Totals older than the last match: its yellow is added (4 + 1 = 5, not suspended; 2 + 1 = 3 would be).
	season.AsOf = "2026-09-13"
	season.Bookings[2].Yellow = 2
	last.Cards = []Card{{Side: "away", Player: "Raphael Veiga", Color: "yellow"}}
	if a := NextMatchAvailability("c", last, season); len(a.Suspended) != 1 || a.Suspended[0].Player != "Raphael Veiga" || a.Note == "" {
		t.Fatalf("stale totals %+v", a)
	}
	// Unknown cards must not claim that nobody is suspended.
	last.Cards = nil
	if a := NextMatchAvailability("c", last, season); len(a.Suspended) != 0 || a.Note == "" {
		t.Fatalf("missing cards %+v", a)
	}
	if NextMatchAvailability("c", last, nil) != nil {
		t.Fatal("no season data must mean no availability")
	}
}

func TestSuspensionsWeakenTheSideMissingRegulars(t *testing.T) {
	in := simInput(t)
	base, _ := Simulate(in, 500)
	in.ClubAvailability = &Availability{Suspended: []Absence{{Player: "Star", MinutesPct: 95, OnOff: 1.2}}}
	in.OpponentAvailability = &Availability{Suspended: []Absence{}}
	weaker, _ := Simulate(in, 500)
	var f SimulationFactor
	for _, x := range weaker.Factors {
		if x.Key == "suspensions" {
			f = x
		}
	}
	if !f.Available || f.Club >= 1 || f.Opponent <= 1 || weaker.ExpectedClub >= base.ExpectedClub {
		t.Fatalf("suspension factor %+v, xG %v vs %v", f, weaker.ExpectedClub, base.ExpectedClub)
	}
}

func TestNextMatchAvailabilityMatchesNamesAcrossSources(t *testing.T) {
	season := &SeasonStats{AsOf: "2026-09-20", Bookings: []PlayerBookings{{Player: "Agustín Giay", Yellow: 3}}}
	last := &Match{Date: "2026-09-20", HomeTeam: Team{ID: "c"}, AwayTeam: Team{ID: "r", Name: "Rival"}, Cards: []Card{{Side: "home", Player: "Agustin Giay", Color: "yellow"}}}
	if a := NextMatchAvailability("c", last, season); len(a.Suspended) != 1 || a.Suspended[0].Player != "Agustín Giay" {
		t.Fatalf("accent-insensitive match failed: %+v", a)
	}
}
