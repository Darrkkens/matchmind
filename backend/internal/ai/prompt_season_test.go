package ai

import (
	"strings"
	"testing"

	"matchmind/internal/football"
)

func TestBuildContextSeasonStats(t *testing.T) {
	snap := &football.Snapshot{
		Team:               &football.Team{ID: "a", Name: "Alpha"},
		SeasonStats:        &football.SeasonStats{Source: "FBref (cópia local)", AsOf: "2026-09-20", Matches: 28, Metrics: []football.SeasonMetric{{Key: "goals", Value: 1.61, League: 1.29, Rank: 2}}},
		NextOpponentSeason: &football.SeasonStats{Team: "Beta", AsOf: "2026-09-20", Metrics: []football.SeasonMetric{{Key: "goals_against", Value: 1.1, League: 1.3, Rank: 6}}},
	}
	ctx, err := BuildContext(snap, "Como está o ataque?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx, `"season_stats"`) || !strings.Contains(ctx, `"goals":{"value":1.61,"league_average":1.29,"rank":2}`) || strings.Contains(ctx, "next_opponent_season") {
		t.Fatalf("attack context: %s", ctx)
	}
	// The opponent's numbers only travel with next-match questions.
	ctx, _ = BuildContext(snap, "Como chega o próximo adversário?")
	if !strings.Contains(ctx, `"next_opponent_season"`) {
		t.Fatalf("next-match context: %s", ctx)
	}
}
