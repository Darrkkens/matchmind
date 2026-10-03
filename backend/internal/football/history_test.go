package football

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Invented CSV in the dataset's format, used only in tests.
const historyFixture = `"ID","rodata","data","hora","mandante","visitante","formacao_mandante","formacao_visitante","tecnico_mandante","tecnico_visitante","vencedor","arena","mandante_Placar","visitante_Placar","mandante_Estado","visitante_Estado"
"1","1","10/05/2019","16:00","Palmeiras","Sao Paulo","4-3-3","","Técnico A","","Palmeiras","Allianz","2","0","SP","SP"
"2","2","17/05/2019","16:00","Sao Paulo","Palmeiras","","4-4-2","","Técnico B","Palmeiras","C","0","1","SP","SP"
"3","1","10/02/2021","16:00","Gremio Prudente","Gremio","","","","","-","B","1","1","SP","RS"
"4","2","20/02/2021","16:00","Gremio","Gremio Prudente","","","","","Gremio","E","1","0","RS","SP"
"5","1","25/05/2021","16:00","Athletico-PR","Atletico-MG","","","","","-","D","0","0","PR","MG"
"6","1","30/05/2021","16:00","Palmeiras","Sao Paulo","","","","","-","A","1","1","SP","SP"
`

const historyGoalsFixture = `"partida_id","rodata","clube","atleta","minuto","tipo_de_gol"
"1","1","Palmeiras","Atacante","10","Penalty"
"1","1","Palmeiras","Atacante","80",""
"2","2","Palmeiras","Meia","45+2",""
"2","2","Palmeiras","Zagueiro Rival","50","Gol Contra"
"999","1","Palmeiras","Fantasma","1",""
`

const historyCardsFixture = `"partida_id","rodata","clube","cartao","atleta","num_camisa","posicao","minuto"
"1","1","Palmeiras","Amarelo","Volante","5","Meio-campo","30"
"2","2","Palmeiras","Vermelho","Volante","5","Meio-campo","70"
"6","1","Palmeiras","Amarelo","Lateral","6","Defensor","12"
"6","1","Sao Paulo","Amarelo","Rival","8","Meio-campo","20"
`

func TestParseHistorySeasonsChampionsAndNames(t *testing.T) {
	data, err := parseHistory(historyFixture, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if data.first != 2019 || data.last != 2021 || data.matches[0].date != "2021-05-30" {
		t.Fatalf("seasons %d-%d first=%+v", data.first, data.last, data.matches[0])
	}
	// February 2021 belongs to the 2020 season; the incomplete 2021 season has no champion.
	if !reflect.DeepEqual(data.champions["Palmeiras"], []int{2019}) || !reflect.DeepEqual(data.champions["Gremio"], []int{2020}) || len(data.champions) != 2 {
		t.Fatalf("champions %v", data.champions)
	}
	for open, want := range map[string]string{"Grêmio FBPA": "Gremio", "SE Palmeiras": "Palmeiras", "São Paulo FC": "Sao Paulo", "CA Paranaense": "Athletico-PR", "CA Mineiro": "Atletico-MG", "Mirassol FC": ""} {
		if got := data.datasetClub(open); got != want {
			t.Errorf("%s mapped to %q, want %q", open, got, want)
		}
	}
	for _, bad := range []string{"", `"data","mandante"` + "\n", strings.Replace(historyFixture, `"Allianz","2","0"`, `"Allianz","x","0"`, 1), strings.Replace(historyFixture, "10/05/2019", "2019-05-10", 1)} {
		if _, err := parseHistory(bad, time.Unix(0, 0)); !errors.Is(err, ErrHistoryUnavailable) {
			t.Fatalf("accepted invalid CSV: %v", err)
		}
	}
}

func historyServer(t *testing.T, handler http.HandlerFunc) *HistoryProvider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	h := NewHistoryProvider()
	h.url = server.URL + "/history.csv"
	return h
}

func TestHistoryInSnapshot(t *testing.T) {
	var calls atomic.Int32
	h := historyServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, "/history.csv"):
			_, _ = w.Write([]byte(historyFixture))
		case strings.HasSuffix(r.URL.Path, "-gols.csv"):
			_, _ = w.Write([]byte(historyGoalsFixture))
		case strings.HasSuffix(r.URL.Path, "-cartoes.csv"):
			_, _ = w.Write([]byte(historyCardsFixture))
		default:
			w.WriteHeader(404) // statistics missing: the rest must still work
		}
	})
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetHistory(h)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	hist := snapshot.History
	if len(hist.Seasons) != 2 || hist.Seasons[0].Season != 2021 || hist.Seasons[1].Position != 1 || hist.Seasons[1].Points != 6 || !reflect.DeepEqual(hist.Seasons[1].Coaches, []HistoryCoach{{"Técnico A", 1}, {"Técnico B", 1}}) || hist.Seasons[1].Stadium != "Allianz" || hist.Seasons[1].Formation != "4-3-3" {
		t.Fatalf("seasons %+v", hist.Seasons)
	}
	if !reflect.DeepEqual(hist.TopScorers, []HistoryScorer{{Player: "Atacante", Goals: 2, Penalties: 1, Seasons: "2019"}, {Player: "Meia", Goals: 1, Seasons: "2019"}}) {
		t.Fatalf("scorers %+v", hist.TopScorers)
	}
	if hist.Discipline == nil || hist.Discipline.Yellow != 2 || hist.Discipline.Red != 1 || hist.Discipline.MostBooked[0] != (HistoryBooking{Player: "Volante", Yellow: 1, Red: 1}) {
		t.Fatalf("discipline %+v", hist.Discipline)
	}
	if !reflect.DeepEqual(hist.Unavailable, []string{"estatisticas-full"}) {
		t.Fatalf("unavailable %v", hist.Unavailable)
	}
	meet := hist.HeadToHead[0].Meetings
	if len(meet) != 3 || meet[2].Stadium != "Allianz" || len(meet[2].Goals) != 2 || meet[2].Goals[0].Player != "Atacante" || meet[2].Goals[0].Kind != "penalty" || len(meet[1].Goals) != 0 {
		t.Fatalf("meetings %+v", meet)
	}
	if hist == nil || hist.DatasetName != "Palmeiras" || hist.SeasonsPlayed != 2 || !reflect.DeepEqual(hist.Titles, []int{2019}) {
		t.Fatalf("history %+v", hist)
	}
	if hist.AllTime != (Record{Played: 3, Wins: 2, Draws: 1, Losses: 0, GoalsFor: 4, GoalsAgainst: 1}) {
		t.Fatalf("all-time %+v", hist.AllTime)
	}
	if len(hist.HeadToHead) != 1 || hist.HeadToHead[0].OpponentName != "São Paulo FC" || hist.HeadToHead[0].Record != (Record{Played: 3, Wins: 2, Draws: 1, Losses: 0, GoalsFor: 4, GoalsAgainst: 1}) || hist.HeadToHead[0].LastMatch.Date != "2021-05-30" {
		t.Fatalf("h2h %+v", hist.HeadToHead)
	}
	if len(snapshot.Trophies) != 1 || snapshot.Trophies[0].Count != 1 || snapshot.Trophies[0].Competition != "Brasileirão Série A (2019–2021)" {
		t.Fatalf("trophies %+v", snapshot.Trophies)
	}
	for _, field := range snapshot.DataMetadata.UnavailableFields {
		if field == "trophies" {
			t.Fatal("trophies still marked unavailable")
		}
	}
	snapshot.History.Titles[0] = 1900
	again, _ := p.GetSnapshot(context.Background(), teams[0].ID, 3)
	if again.History.Titles[0] != 2019 || calls.Load() != 4 {
		t.Fatalf("history cache mutated or refetched (%d calls)", calls.Load())
	}
}

func TestHistoryFailureKeepsSnapshot(t *testing.T) {
	var calls atomic.Int32
	h := historyServer(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(503) })
	p := fixtureProvider(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(openFixture)) })
	p.SetHistory(h)
	teams, _ := p.SearchTeam(context.Background(), "Palmeiras")
	for i := 0; i < 2; i++ {
		snapshot, err := p.GetSnapshot(context.Background(), teams[0].ID, 3)
		if err != nil || snapshot.History != nil || len(snapshot.Trophies) != 0 || snapshot.DataMetadata.HistoryNotice == "" {
			t.Fatalf("%v %+v", err, snapshot)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("missing failure cooldown")
	}
}

func TestSeasonAveragesRequireFilledShotsOnTarget(t *testing.T) {
	d := &historyDataset{stats: map[string]map[string]historyStats{}, tables: map[int][]Standing{}, goals: map[string][]historyGoal{}}
	for i := 0; i < 12; i++ {
		id := fmt.Sprint(i)
		d.matches = append(d.matches, historyMatch{id: id, season: 2016, date: "2016-06-01", home: "A", away: "B", homeScore: 1})
		d.matches = append(d.matches, historyMatch{id: "x" + id, season: 2017, date: "2017-06-01", home: "A", away: "B", homeScore: 1})
		d.stats[id] = map[string]historyStats{"A": {shots: 10, onTarget: 0, possession: 50, corners: 4, fouls: 12}}
		d.stats["x"+id] = map[string]historyStats{"A": {shots: 10, onTarget: 4, possession: 55, corners: 6, fouls: 10}}
	}
	seasons := d.seasons("A")
	if len(seasons) != 2 || seasons[0].Averages == nil || seasons[0].Averages.ShotsOnTarget != 4 || seasons[1].Averages != nil {
		t.Fatalf("%+v", seasons)
	}
}
