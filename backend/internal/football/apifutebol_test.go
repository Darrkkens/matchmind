package football

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Invented API-Futebol responses, used only in tests.
const futebolRound3 = `{"status":"encerrada","partidas":[
{"partida_id":900,"time_mandante":{"time_id":1,"nome_popular":"Palmeiras"},"time_visitante":{"time_id":2,"nome_popular":"São Paulo"},"placar_mandante":2,"placar_visitante":0,"status":"finalizado"}]}`
const futebolRound2 = `{"status":"encerrada","partidas":[
{"partida_id":800,"time_mandante":{"time_id":2,"nome_popular":"São Paulo"},"time_visitante":{"time_id":1,"nome_popular":"Palmeiras"},"placar_mandante":3,"placar_visitante":3,"status":"finalizado"}]}`
const futebolDetail900 = `{"partida_id":900,"time_mandante":{"time_id":1,"nome_popular":"Palmeiras"},"time_visitante":{"time_id":2,"nome_popular":"São Paulo"},"placar_mandante":2,"placar_visitante":0,"status":"finalizado",
"estatisticas":{"mandante":{"posse_de_bola":"57%","escanteios":1,"faltas":9,"finalizacao":{"total":5,"no_gol":1}},"visitante":{"posse_de_bola":"43%","escanteios":6,"faltas":12,"finalizacao":{"total":8,"no_gol":5}}},
"cartoes":{"amarelo":{"mandante":[{},{}],"visitante":[{}]},"vermelho":{"mandante":[],"visitante":[{}]}}}`

// Rounds follow openFixture: 1 (0-1), 2 (1-1), 3 (2-0).
func futebolServer(t *testing.T, calls *atomic.Int32) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer live_key" && r.Header.Get("Authorization") != "Bearer test_key" {
			t.Errorf("auth header %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/campeonatos/10/rodadas/3":
			_, _ = w.Write([]byte(futebolRound3))
		case "/campeonatos/10/rodadas/2":
			_, _ = w.Write([]byte(futebolRound2))
		case "/partidas/900", "/partidas/800":
			_, _ = w.Write([]byte(futebolDetail900))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestAPIFutebolEnrichesAndCachesForever(t *testing.T) {
	var calls atomic.Int32
	server := futebolServer(t, &calls)
	cache := NewMemoryCache()
	newProvider := func() *OpenFootballProvider {
		stats, err := NewAPIFutebolStats("live_key", "br.1", cache)
		if err != nil {
			t.Fatal(err)
		}
		stats.root = server.URL
		p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
		p.SetStatistics(stats)
		return p
	}
	p := newProvider()
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	m := snapshot.RecentMatches
	want := MatchStatistics{Possession: StatPair{57, 43}, Shots: StatPair{5, 8}, ShotsOnTarget: StatPair{1, 5}, Corners: StatPair{1, 6}, Fouls: StatPair{9, 12}, YellowCards: StatPair{2, 1}, RedCards: StatPair{0, 1}}
	if m[0].Statistics == nil || *m[0].Statistics != want {
		t.Fatalf("stats %+v", m[0].Statistics)
	}
	// Round 2 reports 3-3 while OpenFootball has 1-1: never attach mismatched data.
	if m[1].Statistics != nil || m[2].Statistics != nil {
		t.Fatal("mismatched or unknown matches must keep nil statistics")
	}
	if snapshot.DataMetadata.StatisticsSource != "api-futebol" {
		t.Fatalf("metadata %+v", snapshot.DataMetadata)
	}
	first := calls.Load()
	// A new process sharing the cache (as with PostgreSQL) makes no new requests.
	p = newProvider()
	teams, _ = p.SearchTeam(context.Background(), "Palmeiras")
	if _, err := p.GetSnapshot(context.Background(), teams[0].ID, 3); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != first {
		t.Fatalf("cached responses re-requested: %d -> %d", first, calls.Load())
	}
}

func TestAPIFutebolTestKeyIsLabeled(t *testing.T) {
	var calls atomic.Int32
	server := futebolServer(t, &calls)
	stats, _ := NewAPIFutebolStats("test_key", "br.1", nil)
	stats.root = server.URL
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetStatistics(stats)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	meta := snapshot.DataMetadata
	if meta.StatisticsSource != "api-futebol-teste" || !strings.Contains(meta.StatisticsNotice, "exemplo") {
		t.Fatalf("test data not labeled: %+v", meta)
	}
}

func TestAPIFutebolConfigAndNames(t *testing.T) {
	for _, tt := range [][2]string{{"", "br.1"}, {"a b", "br.1"}, {"key", "en.1"}} {
		if _, err := NewAPIFutebolStats(tt[0], tt[1], nil); err == nil {
			t.Fatalf("accepted %v", tt)
		}
	}
	for _, pair := range [][2]string{{"CA Mineiro", "Atlético-MG"}, {"CA Paranaense", "Athletico-PR"}, {"Clube do Remo", "Remo"}, {"CR Vasco da Gama", "Vasco"}, {"RB Bragantino", "Bragantino"}, {"Grêmio FBPA", "Grêmio"}} {
		if !sameBrazilianClub(pair[0], pair[1]) {
			t.Errorf("%s should match %s", pair[0], pair[1])
		}
	}
	if sameBrazilianClub("CA Mineiro", "Athletico-PR") || sameBrazilianClub("CA Paranaense", "Atlético-MG") {
		t.Fatal("Atléticos collided")
	}
}

func TestAPIFutebolPlanErrorIsShown(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"message":"Este campeonato não faz parte do seu plano.","code":401}`))
	}))
	t.Cleanup(server.Close)
	stats, _ := NewAPIFutebolStats("live_key", "br.1", nil)
	stats.root = server.URL
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetStatistics(stats)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	for i := 0; i < 2; i++ {
		snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
		if err != nil || len(snapshot.RecentMatches) != 3 || !strings.Contains(snapshot.DataMetadata.StatisticsNotice, "não faz parte do seu plano") {
			t.Fatalf("%v %+v", err, snapshot.DataMetadata)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("plan error retried %d times; expected a cooldown", calls.Load())
	}
}
