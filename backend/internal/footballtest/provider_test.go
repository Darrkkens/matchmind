package footballtest

import (
	"context"
	"matchmind/internal/football"
	"reflect"
	"testing"
)

func TestDemoRecentForm(t *testing.T) {
	m, err := NewProvider().GetRecentMatches(context.Background(), "demo-palmeiras", 3)
	if err != nil {
		t.Fatal(err)
	}
	f := football.CalculateForm("demo-palmeiras", m)
	if f.Played != 3 || f.Wins != 2 || f.Draws != 1 || f.Losses != 0 || f.GoalsScored != 5 || f.GoalsConceded != 2 || f.AverageGoals != 1.67 || f.PointsPercentage != 77.8 || !reflect.DeepEqual(f.Sequence, []string{"W", "D", "W"}) {
		t.Fatalf("unexpected form %+v", f)
	}
}
func TestDemoLookupAndCancellation(t *testing.T) {
	p := NewProvider()
	teams, err := p.SearchTeam(context.Background(), "pAlMeIrAs")
	if err != nil || len(teams) != 1 {
		t.Fatalf("%v %v", teams, err)
	}
	for _, limit := range []int{-1, 0, 1, 3, 9} {
		m, err := p.GetRecentMatches(context.Background(), teams[0].ID, limit)
		if err != nil || len(m) > 3 || (limit >= 0 && len(m) > limit) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.SearchTeam(ctx, "Palmeiras"); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}
