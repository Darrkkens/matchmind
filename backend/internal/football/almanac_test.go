package football

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// Invented responses in the AlmanacStats format, used only in tests. Rounds follow
// openFixture: 1 Palmeiras 0-1 São Paulo, 2 São Paulo 1-1 Palmeiras, 3 Palmeiras 2-0.
const almanacList = `{"data":[{"home":{"slug":"palmeiras","name":"Palmeiras","score":2},"away":{"slug":"sao-paulo","name":"São Paulo","score":0}}]}`
const almanacProfileBody = `{"data":{"club":{"league":{"slug":"serie-a-brazil"},
"matches":[
{"mid":3,"status":"FT","round":"Regular Season - 3","side":"home","home_score":2,"away_score":0,"league":"Brasileirão Série A","opponent":{"slug":"sao-paulo","name":"São Paulo"}},
{"mid":2,"status":"FT","round":"Regular Season - 2","side":"away","home_score":5,"away_score":5,"league":"Brasileirão Série A","opponent":{"slug":"sao-paulo","name":"São Paulo"}},
{"mid":9,"status":"FT","round":"Group Stage - 1","side":"home","home_score":1,"away_score":0,"league":"CONMEBOL Libertadores","opponent":{"slug":"x","name":"Outro"}}],
"squad":[{"slug":"g1","name":"Goleiro Um","position":"Goalkeeper","apps":30,"goals":0,"assists":0,"rating":"6.9"},{"slug":"a1","name":"Atacante Um","position":"Attack","apps":28,"goals":12,"assists":4,"rating":"7.4"}]}}}`
const almanacDetail3 = `{"data":{"status":"FT","home":{"slug":"palmeiras","name":"Palmeiras","score":2},"away":{"slug":"sao-paulo","name":"São Paulo","score":0},
"venue":{"name":"Allianz Parque","city":"São Paulo"},
"lineups":{"home":{"formation":"3-4-2-1","coach":"Técnico Casa","start":[{"pos":"G","name":"Goleiro Um","number":1},{"pos":"F","name":"Atacante Um","number":9}],"subs":[{"pos":"M","name":"Reserva","number":20}]},
"away":{"formation":"4-4-2","coach":"Técnico Fora","start":[{"pos":"D","name":"Zagueiro Rival","number":4}],"subs":[]}},
"lineup_home":[{"name":"Atacante Um Silva","position":"Attack","minutes":90,"rating":"7.9","goals":1,"assists":0,"yellow":0,"red":0},{"name":"Goleiro Um","position":"Goalkeeper","minutes":90,"rating":"6.8","goals":0,"assists":0,"yellow":1,"red":0}],
"lineup_away":[],"referee":"Árbitro Teste, Brazil",
"events":[{"side":"home","type":"Goal","detail":"Normal Goal","minute":50,"player":"Atacante Um","assist":"Meia Dois"},
{"side":"away","type":"Var","detail":"Goal cancelled","minute":60,"player":"Anulado","assist":null},
{"side":"home","type":"Goal","detail":"Own Goal","minute":"45+1","player":"Zagueiro Rival","assist":"Ignorada"},
{"side":"home","type":"Card","detail":"Yellow Card","minute":10,"player":"Alguém","assist":null}],
"team_stats":{"home":{"possession":"66%","shots":11,"shots_on":6,"corners":2,"fouls":9,"yellow":3,"red":0},"away":{"possession":"34%","shots":15,"shots_on":5,"corners":5,"fouls":15,"yellow":3,"red":1}}}}`

func almanacServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/v1/matches":
			if r.URL.Query().Get("league") != "serie-a-brazil" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(almanacList))
		case "/v1/team/palmeiras/profile":
			_, _ = w.Write([]byte(almanacProfileBody))
		case "/v1/match/3":
			_, _ = w.Write([]byte(almanacDetail3))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"statusCode":404}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestAlmanacStatsEnrichesSquadAndCaches(t *testing.T) {
	var calls atomic.Int32
	server := almanacServer(t, &calls)
	cache := NewMemoryCache()
	build := func() *OpenFootballProvider {
		a := NewAlmanacStats(cache)
		a.root, a.interval = server.URL, time.Millisecond
		p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
		p.SetStatistics(a)
		p.SetSquad(a)
		return p
	}
	p := build()
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	m := snapshot.RecentMatches
	want := MatchStatistics{Possession: StatPair{66, 34}, Shots: StatPair{11, 15}, ShotsOnTarget: StatPair{6, 5}, Corners: StatPair{2, 5}, Fouls: StatPair{9, 15}, YellowCards: StatPair{3, 3}, RedCards: StatPair{0, 1}}
	if m[0].Statistics == nil || *m[0].Statistics != want {
		t.Fatalf("stats %+v", m[0].Statistics)
	}
	wantGoals := []Goal{{Side: "home", Minute: "45+1", Player: "Zagueiro Rival", Kind: "own_goal"}, {Side: "home", Minute: "50", Player: "Atacante Um", Assist: "Meia Dois", Kind: "normal"}}
	if m[0].Venue != "Allianz Parque (São Paulo)" || m[0].Referee != "Árbitro Teste" || len(m[0].Cards) != 1 || m[0].Cards[0] != (Card{Side: "home", Minute: "10", Player: "Alguém", Color: "yellow"}) {
		t.Fatalf("venue/referee/cards %q %q %+v", m[0].Venue, m[0].Referee, m[0].Cards)
	}
	if m[0].LineupRef != "3" || m[1].LineupRef != "" {
		t.Fatalf("lineup refs %q %q", m[0].LineupRef, m[1].LineupRef)
	}
	if !reflect.DeepEqual(m[0].Goals, wantGoals) {
		t.Fatalf("goals %+v", m[0].Goals)
	}
	// Round 2 reports 5-5 on AlmanacStats but 1-1 on OpenFootball: never attach it.
	if m[1].Statistics != nil || m[2].Statistics != nil {
		t.Fatal("mismatched or missing matches must keep nil statistics")
	}
	if len(snapshot.Squad) != 2 || snapshot.Squad[1] != (Player{ID: "al-a1", Name: "Atacante Um", Position: "Atacante", Appearances: 28, Goals: 12, Assists: 4, Rating: 7.4}) {
		t.Fatalf("squad %+v", snapshot.Squad)
	}
	meta := snapshot.DataMetadata
	if meta.StatisticsSource != "almanacstats" {
		t.Fatalf("metadata %+v", meta)
	}
	for _, field := range meta.UnavailableFields {
		if field == "squad" {
			t.Fatal("squad still marked unavailable")
		}
	}
	first := calls.Load()
	if first != 3 {
		t.Fatalf("expected list, profile and one match detail, got %d requests", first)
	}
	p = build()
	teams, _ = p.SearchTeam(context.Background(), "Palmeiras")
	if _, err := p.GetSnapshot(context.Background(), teams[0].ID, 3); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != first {
		t.Fatalf("cached responses re-requested: %d -> %d", first, calls.Load())
	}
}

func TestAlmanacStatsRateLimit(t *testing.T) {
	var calls atomic.Int32
	server := almanacServer(t, &calls)
	a := NewAlmanacStats(nil)
	a.root, a.interval = server.URL, 80*time.Millisecond
	start := time.Now()
	var out any
	for _, path := range []string{"/v1/team/palmeiras/profile", "/v1/match/3", "/v1/matches?league=serie-a-brazil"} {
		if err := a.get(context.Background(), path, &out, fixedTTL(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 160*time.Millisecond {
		t.Fatalf("requests not spaced: %v", elapsed)
	}
}

func TestAlmanacNameMatching(t *testing.T) {
	for _, pair := range [][2]string{{"CA Mineiro", "Atlético Mineiro"}, {"CA Paranaense", "Athletico Paranaense"}, {"CR Vasco da Gama", "Vasco da Gama"}, {"Grêmio FBPA", "Grêmio"}, {"SC Corinthians Paulista", "Corinthians"}, {"RB Bragantino", "RB Bragantino"}} {
		if _, ok := nameDistance(pair[0], pair[1]); !ok {
			t.Errorf("%s should match %s", pair[0], pair[1])
		}
	}
	if _, ok := nameDistance("CA Mineiro", "Athletico Paranaense"); ok {
		t.Fatal("Atléticos collided")
	}
	for _, slug := range []string{"", "../etc", "a/b", "UPPER", "with space"} {
		if validSlug(slug) {
			t.Fatalf("accepted slug %q", slug)
		}
	}
}

func TestAlmanacGoalsMustMatchScore(t *testing.T) {
	d := almanacDetail{Events: []almanacEvent{{Side: "home", Type: "Goal", Detail: "Penalty", Minute: []byte(`"90+3"`), Player: "Batedor"}}}
	if got := d.goals(1, 0); len(got) != 1 || got[0].Kind != "penalty" || got[0].Minute != "90+3" {
		t.Fatalf("goals %+v", got)
	}
	d.Events = append(d.Events, almanacEvent{Side: "away", Type: "Card", Detail: "Red Card", Minute: []byte(`88`), Player: "Expulso"}, almanacEvent{Side: "home", Type: "Card", Detail: "Yellow Card", Minute: []byte(`"45+1"`), Player: "Advertido"}, almanacEvent{Side: "away", Type: "Var", Detail: "Card upgrade confirmed", Minute: []byte(`87`), Player: "Expulso"})
	if got := d.cards(); !reflect.DeepEqual(got, []Card{{Side: "home", Minute: "45+1", Player: "Advertido", Color: "yellow"}, {Side: "away", Minute: "88", Player: "Expulso", Color: "red"}}) {
		t.Fatalf("cards %+v", got)
	}
	if d.goals(2, 0) != nil || d.goals(1, 1) != nil {
		t.Fatal("incomplete goal list attached")
	}
	if minuteValue("45+2") <= minuteValue("45") || minuteValue("45+2") >= minuteValue("46") {
		t.Fatal("stoppage time ordering")
	}
}

func TestAlmanacLineups(t *testing.T) {
	var calls atomic.Int32
	server := almanacServer(t, &calls)
	a := NewAlmanacStats(nil)
	a.root, a.interval = server.URL, time.Millisecond
	got, err := a.Lineups(context.Background(), "3")
	if err != nil {
		t.Fatal(err)
	}
	home := got.Home
	if home.Formation != "3-4-2-1" || home.Coach != "Técnico Casa" || len(home.Starters) != 2 || len(home.Substitutes) != 1 || got.Away.Coach != "Técnico Fora" {
		t.Fatalf("lineups %+v", got)
	}
	want := LineupPlayer{Number: 9, Name: "Atacante Um", Position: "Atacante", Minutes: 90, Rating: 7.9, Goals: 1}
	if home.Starters[1] != want || home.Starters[0].Yellow != 1 || home.Substitutes[0].Minutes != 0 {
		t.Fatalf("players %+v", home)
	}
	for _, ref := range []string{"abc", "-1", "0", "99"} {
		if _, err := a.Lineups(context.Background(), ref); !errors.Is(err, ErrLineupsUnavailable) {
			t.Fatalf("ref %q: %v", ref, err)
		}
	}
}

func TestPlayerDistance(t *testing.T) {
	for _, pair := range [][2]string{{"G. Gómez", "Gustavo Gómez"}, {"A. Giay", "Agustin Giay"}, {"Osorio Luis", "Luis Osorio Messias De Oliveira"}, {"Pedro", "Pedro"}, {"R. Tolói", "Rafael Tolói"}} {
		if _, ok := playerDistance(pair[0], pair[1]); !ok {
			t.Errorf("%s should match %s", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]string{{"G. Gómez", "Gabriel Silva"}, {"J. López", "Jorge Lima"}, {"A. B.", "Ana Bia"}} {
		if _, ok := playerDistance(pair[0], pair[1]); ok {
			t.Errorf("%s must not match %s", pair[0], pair[1])
		}
	}
}

func TestCleanName(t *testing.T) {
	for in, want := range map[string]string{"A. Sant&apos;Anna": "A. Sant'Anna", "JoÃ£o Pedro": "João Pedro", "Vitão": "Vitão", " Pedro ": "Pedro", "SÃ£o": "São"} {
		if got := cleanName(in); got != want {
			t.Errorf("cleanName(%q) = %q, want %q", in, got, want)
		}
	}
}
