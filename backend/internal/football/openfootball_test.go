package football

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Invented HTTP fixture used only in tests, never exposed as real application data.
const openFixture = `{"name":"Test League 2026","matches":[
{"round":"1","date":"2026-09-01","team1":"SE Palmeiras","team2":"São Paulo FC","score":{"ft":[0,1]}},
{"round":"2","date":"2026-09-15","team1":"São Paulo FC","team2":"SE Palmeiras","score":{"ft":[1,1]}},
{"round":"3","date":"2026-09-30","team1":"SE Palmeiras","team2":"São Paulo FC","score":{"ft":[2,0]}},
{"round":"4","date":"2026-10-01","team1":"SE Palmeiras","team2":"São Paulo FC"},
{"round":"5","date":"2026-12-01","team1":"SE Palmeiras","team2":"São Paulo FC","score":{"ft":[9,9]}}
]}`

func fixtureProvider(t *testing.T, handler http.HandlerFunc) *OpenFootballProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	p, err := NewOpenFootballProvider("br.1", "2026")
	if err != nil {
		t.Fatal(err)
	}
	p.url = server.URL + "/2026/br.1.json"
	p.now = func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }
	return p
}

func TestOpenFootballResultsAndCoverage(t *testing.T) {
	var calls atomic.Int32
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/2026/br.1.json" || r.Method != "GET" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(openFixture))
	})
	ctx := context.Background()
	teams, err := p.SearchTeam(ctx, "pAlMeIrAs")
	if err != nil || len(teams) != 1 {
		t.Fatalf("%v %v", teams, err)
	}
	team := teams[0]
	if team.Coach != "" || team.FoundedYear != 0 {
		t.Fatal("profile facts must remain unknown")
	}
	if team.LogoURL != "/crests/palmeiras.svg" {
		t.Fatal("missing curated crest")
	}
	for _, alias := range []string{"Verdão", "verdao"} {
		if found, err := p.SearchTeam(ctx, alias); err != nil || len(found) != 1 || found[0].ID != team.ID {
			t.Fatalf("alias %s: %v %v", alias, found, err)
		}
	}
	accented, err := p.SearchTeam(ctx, "sao-paulo")
	if err != nil || len(accented) != 1 || accented[0].Name != "São Paulo FC" {
		t.Fatalf("accent lookup: %v %v", accented, err)
	}
	snapshot, err := p.GetSnapshot(ctx, team.ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	f := snapshot.RecentForm
	if f.Played != 3 || f.Wins != 1 || f.Draws != 1 || f.Losses != 1 || f.GoalsScored != 3 || f.GoalsConceded != 2 || f.PointsPercentage != 44.4 || !reflect.DeepEqual(f.Sequence, []string{"W", "D", "L"}) {
		t.Fatalf("form %+v", f)
	}
	if snapshot.DataSource != "openfootball" || snapshot.DataMetadata.Season != "2026" || snapshot.DataMetadata.LatestMatchDate != "2026-09-30" || snapshot.DataMetadata.FetchedAt != "2026-10-02T12:00:00Z" {
		t.Fatalf("metadata %+v", snapshot)
	}
	if len(snapshot.Squad) != 0 || snapshot.Squad == nil || len(snapshot.Trophies) != 0 || snapshot.Trophies == nil {
		t.Fatal("missing sections must be empty arrays")
	}
	for _, m := range snapshot.RecentMatches {
		if m.Statistics != nil {
			t.Fatal("missing stats must stay nil")
		}
	}
	// Palmeiras and São Paulo tie on points and wins; goal difference (+1 vs -1) decides.
	if len(snapshot.Standings) != 2 || snapshot.Standings[0].TeamID != team.ID || snapshot.Standings[0].Points != 4 || snapshot.Standings[0].Played != 3 || snapshot.Standings[1].GoalDifference != -1 {
		t.Fatalf("standings %+v", snapshot.Standings)
	}
	table, err := p.GetStandings(ctx)
	if err != nil || table.Season != "2026" || table.Competition != "Test League 2026" || !reflect.DeepEqual(table.Standings, snapshot.Standings) {
		t.Fatalf("table %+v %v", table, err)
	}
	// Returned values must not mutate the shared cache used by later chat requests.
	snapshot.Standings[0].Points = 99
	snapshot.Team.Name = "tampered"
	snapshot.RecentMatches[0].HomeScore = 100
	unchanged, _ := p.GetSnapshot(ctx, team.ID, 3)
	if unchanged.Team.Name != team.Name || unchanged.RecentForm.GoalsScored != 3 || unchanged.Standings[0].Points != 4 {
		t.Fatal("cache is externally mutable")
	}
	for _, limit := range []int{0, -1, 1, 100} {
		matches, err := p.GetRecentMatches(ctx, team.ID, limit)
		if err != nil || len(matches) > 3 || (limit <= 0 && len(matches) != 0) || (limit == 1 && len(matches) != 1) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
	if _, err := p.GetTeam(ctx, "demo-palmeiras"); !errors.Is(err, ErrNotFound) {
		t.Fatal("demo ID accepted by real provider")
	}
	if _, err := p.GetSquad(ctx, team.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetTrophies(ctx, team.ID); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one cached download, got %d", calls.Load())
	}
}

func TestOpenFootballRejectsInvalidDataAndHTTPFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"missing season", 404, "", ErrDatasetUnavailable},
		{"rate limit", 429, "", ErrProviderUnavailable},
		{"upstream down", 503, "", ErrProviderUnavailable},
		{"invalid json", 200, "oops", ErrProviderData},
		{"error envelope with HTTP200", 200, `{"error":"rate limited"}`, ErrProviderData},
		{"missing matches", 200, `{"name":"League"}`, ErrProviderData},
		{"null score", 200, strings.Replace(openFixture, `[2,0]`, `[null,0]`, 1), ErrProviderData},
		{"negative score", 200, strings.Replace(openFixture, `[2,0]`, `[-1,0]`, 1), ErrProviderData},
		{"invalid date", 200, strings.Replace(openFixture, "2026-09-30", "2026-99-99", 1), ErrProviderData},
		{"oversized", 200, strings.Repeat("x", maxDatasetBytes+1), ErrProviderData},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})
			for i := 0; i < 2; i++ {
				if _, err := p.SearchTeam(context.Background(), "Palmeiras"); !errors.Is(err, tt.want) {
					t.Fatalf("got %v want %v", err, tt.want)
				}
			}
			if p.cached != nil {
				t.Fatal("invalid data cached as success")
			}
			if calls.Load() != 1 {
				t.Fatal("missing failure cooldown")
			}
		})
	}
}

func TestOpenFootballDoesNotServeExpiredDataAfterFailedRefresh(t *testing.T) {
	var failed atomic.Bool
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if failed.Load() {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write([]byte(openFixture))
	})
	if _, err := p.SearchTeam(context.Background(), "Palmeiras"); err != nil {
		t.Fatal(err)
	}
	failed.Store(true)
	p.ttl = -time.Second
	if _, err := p.SearchTeam(context.Background(), "Palmeiras"); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expired records silently reused: %v", err)
	}
}

func TestOpenFootballConcurrentCacheAndCancellation(t *testing.T) {
	var calls atomic.Int32
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = w.Write([]byte(openFixture)) })
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.SearchTeam(context.Background(), "Palmeiras"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("parallel cache miss downloaded %d times", calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.SearchTeam(ctx, "Palmeiras"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	p.gate <- struct{}{}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer stop()
	if _, err := p.SearchTeam(ctx, "Palmeiras"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	<-p.gate
}

func TestOpenFootballTimeoutAndRedirect(t *testing.T) {
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	})
	p.client.Timeout = 10 * time.Millisecond
	if _, err := p.SearchTeam(context.Background(), "Palmeiras"); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal(err)
	}
	var followed atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed.Store(true) }))
	defer target.Close()
	p = fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) })
	if _, err := p.SearchTeam(context.Background(), "Palmeiras"); !errors.Is(err, ErrProviderUnavailable) || followed.Load() {
		t.Fatal("redirect must not be followed")
	}
}

func TestOpenFootballConfigCannotInjectURL(t *testing.T) {
	for _, tt := range [][2]string{{"http://localhost", "2026"}, {"br.1", "../../etc"}, {"br.1", "2026?url=http://localhost"}, {"unknown", "2026"}} {
		if _, err := NewOpenFootballProvider(tt[0], tt[1]); err == nil {
			t.Fatalf("unsafe config accepted %v", tt)
		}
	}
	if _, err := NewOpenFootballProvider("en.1", "2026-27"); err != nil {
		t.Fatal(err)
	}
}

func TestOpenFootballCompactScoreIncludesScorelessDraw(t *testing.T) {
	body := strings.Replace(openFixture, `{"ft":[2,0]}`, `[0,0]`, 1)
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) })
	teams, err := p.SearchTeam(context.Background(), "Palmeiras")
	if err != nil {
		t.Fatal(err)
	}
	matches, err := p.GetRecentMatches(context.Background(), teams[0].ID, 3)
	if err != nil || len(matches) != 3 {
		t.Fatalf("%v %v", matches, err)
	}
	if matches[0].HomeScore != 0 || matches[0].AwayScore != 0 || matches[0].Date != "2026-09-30" {
		t.Fatal("explicit 0-0 draw was lost")
	}
}
