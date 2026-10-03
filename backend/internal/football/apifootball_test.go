package football

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Invented API-Football responses for tests only.
const apiFixtures = `{"errors":[],"response":[
{"fixture":{"id":11,"date":"2026-09-01T00:30:00+00:00","status":{"short":"FT"}},"teams":{"home":{"id":1,"name":"Palmeiras"},"away":{"id":2,"name":"Sao Paulo"}},"goals":{"home":0,"away":1}},
{"fixture":{"id":12,"date":"2026-09-15T22:00:00+00:00","status":{"short":"FT"}},"teams":{"home":{"id":2,"name":"Sao Paulo"},"away":{"id":1,"name":"Palmeiras"}},"goals":{"home":1,"away":1}},
{"fixture":{"id":13,"date":"2026-09-30T22:00:00+00:00","status":{"short":"FT"}},"teams":{"home":{"id":1,"name":"Palmeiras"},"away":{"id":2,"name":"Sao Paulo"}},"goals":{"home":2,"away":0}}
]}`

const apiStats = `{"errors":[],"response":[
{"team":{"id":1,"name":"Palmeiras"},"statistics":[{"type":"Ball Possession","value":"61%"},{"type":"Total Shots","value":18},{"type":"Shots on Goal","value":7},{"type":"Corner Kicks","value":6},{"type":"Fouls","value":11},{"type":"Yellow Cards","value":2},{"type":"Red Cards","value":null}]},
{"team":{"id":2,"name":"Sao Paulo"},"statistics":[{"type":"Ball Possession","value":"39%"},{"type":"Total Shots","value":9},{"type":"Shots on Goal","value":2},{"type":"Corner Kicks","value":3},{"type":"Fouls","value":13},{"type":"Yellow Cards","value":3},{"type":"Red Cards","value":1}]}
]}`

func statsServer(t *testing.T, handler http.HandlerFunc) *APIFootballStats {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	s, err := NewAPIFootballStats("test-key", "br.1", "2026")
	if err != nil {
		t.Fatal(err)
	}
	s.root = server.URL
	return s
}

func TestAPIFootballEnrichesSnapshotAndCaches(t *testing.T) {
	var fixtureCalls, statCalls atomic.Int32
	stats := statsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-apisports-key") != "test-key" {
			t.Error("missing API key header")
		}
		switch r.URL.Path {
		case "/fixtures":
			fixtureCalls.Add(1)
			if r.URL.Query().Get("league") != "71" || r.URL.Query().Get("season") != "2026" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(apiFixtures))
		case "/fixtures/statistics":
			statCalls.Add(1)
			if r.URL.Query().Get("fixture") == "12" {
				_, _ = w.Write([]byte(`{"errors":[],"response":[]}`))
				return
			}
			_, _ = w.Write([]byte(apiStats))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetStatistics(stats)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	m := snapshot.RecentMatches
	if m[0].Statistics == nil || m[0].Statistics.Possession != (StatPair{61, 39}) || m[0].Statistics.Shots != (StatPair{18, 9}) || m[0].Statistics.RedCards != (StatPair{0, 1}) {
		t.Fatalf("stats %+v", m[0].Statistics)
	}
	if m[1].Statistics != nil || m[2].Statistics == nil {
		t.Fatal("match without provider statistics must stay nil")
	}
	meta := snapshot.DataMetadata
	if meta.StatisticsSource != "api-football" || meta.StatisticsNotice == "" || meta.UnavailableFields[len(meta.UnavailableFields)-1] != "match_statistics" {
		t.Fatalf("metadata %+v", meta)
	}
	if _, err := p.GetSnapshot(context.Background(), teams[0].ID, 3); err != nil {
		t.Fatal(err)
	}
	if fixtureCalls.Load() != 1 || statCalls.Load() != 3 {
		t.Fatalf("expected cached calls, got fixtures=%d stats=%d", fixtureCalls.Load(), statCalls.Load())
	}
}

func TestAPIFootballPlanErrorDoesNotBlockResults(t *testing.T) {
	var calls atomic.Int32
	stats := statsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"errors":{"plan":"Free plans do not have access to this season."},"response":[]}`))
	})
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetStatistics(stats)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	for i := 0; i < 2; i++ {
		snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
		if err != nil || len(snapshot.RecentMatches) != 3 || snapshot.RecentMatches[0].Statistics != nil {
			t.Fatalf("%v %+v", err, snapshot)
		}
		if !strings.Contains(snapshot.DataMetadata.StatisticsNotice, "Free plans") || snapshot.DataMetadata.StatisticsSource != "" {
			t.Fatalf("notice %+v", snapshot.DataMetadata)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("missing failure cooldown")
	}
	stats.lastFailure = nil
	stats.root = "http://127.0.0.1:1"
	if err := stats.Enrich(context.Background(), Team{}, []Match{}); !errors.Is(err, ErrStatsUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestFindFixtureRequiresDateScoreAndClubs(t *testing.T) {
	two, one := 2, 1
	f := apiFixture{}
	f.Fixture.Date = "2026-05-11T00:30:00+00:00"
	f.Teams.Home.Name, f.Teams.Away.Name = "Atletico-MG", "Corinthians"
	f.Goals.Home, f.Goals.Away = &two, &one
	match := Match{Date: "2026-05-10", HomeTeam: Team{Name: "CA Mineiro"}, AwayTeam: Team{Name: "SC Corinthians Paulista"}, HomeScore: 2, AwayScore: 1}
	if findFixture([]apiFixture{f}, match) == nil {
		t.Fatal("expected late-evening fixture to match across UTC midnight")
	}
	for _, changed := range []Match{
		{Date: "2026-05-13", HomeTeam: match.HomeTeam, AwayTeam: match.AwayTeam, HomeScore: 2, AwayScore: 1},
		{Date: match.Date, HomeTeam: match.HomeTeam, AwayTeam: match.AwayTeam, HomeScore: 1, AwayScore: 1},
		{Date: match.Date, HomeTeam: Team{Name: "CA Paranaense"}, AwayTeam: match.AwayTeam, HomeScore: 2, AwayScore: 1},
	} {
		if findFixture([]apiFixture{f}, changed) != nil {
			t.Fatalf("false match %+v", changed)
		}
	}
	pairs := [][2]string{{"Botafogo FR", "Botafogo"}, {"RB Bragantino", "RB Bragantino"}, {"CR Vasco da Gama", "Vasco DA Gama"}, {"Chapecoense AF", "Chapecoense-sc"}, {"Clube do Remo", "Remo"}, {"CA Paranaense", "Atletico Paranaense"}, {"São Paulo FC", "Sao Paulo"}, {"Grêmio FBPA", "Gremio"}}
	for _, pair := range pairs {
		if !sameClub(pair[0], pair[1]) {
			t.Errorf("%s should match %s", pair[0], pair[1])
		}
	}
	if sameClub("CA Mineiro", "Atletico Paranaense") || sameClub("CA Paranaense", "Atletico-MG") || sameClub("SC Internacional", "Gremio") {
		t.Fatal("different clubs matched")
	}
}

func TestAPIFootballConfigValidation(t *testing.T) {
	for _, tt := range [][3]string{{"", "br.1", "2026"}, {"key with space", "br.1", "2026"}, {"k", "en.1", "2026-27"}, {"k", "br.1", "2026-27"}} {
		if _, err := NewAPIFootballStats(tt[0], tt[1], tt[2]); err == nil {
			t.Fatalf("accepted %v", tt)
		}
	}
	s, _ := NewAPIFootballStats("k", "br.1", "2026")
	s.now = func() time.Time { return time.Unix(0, 0) }
	if s.Name() != "api-football" {
		t.Fatal(s.Name())
	}
}
