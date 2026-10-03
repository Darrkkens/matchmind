package football

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// FBrefSeason serves season totals from a local folder of CSV exports of FBref's
// Brasileirão tables (see README). Nothing is downloaded: the folder is read once at
// start-up, and the data is a dated snapshot, never live.
type FBrefSeason struct {
	asOf  string
	clubs map[string]*SeasonStats // keyed by the FBref squad name
}

const fbrefSourceURL = "https://fbref.com/en/comps/24/Serie-A-Stats"
const maxCSVBytes = 8 << 20

// Metric definitions: group, the CSV column (by file) and whether a lower value is better.
type fbrefMetric struct {
	key, group, file, column string
	perMatch, lowerIsBetter  bool
}

var fbrefMetrics = []fbrefMetric{
	{"goals", "attack", "times_padrao", "Performance_Gls", true, false},
	{"shots", "attack", "times_finalizacao", "Standard_Sh", true, false},
	{"shots_on_target", "attack", "times_finalizacao", "Standard_SoT", true, false},
	{"shot_accuracy", "attack", "times_finalizacao", "Standard_SoT%", false, false},
	{"goals_per_shot", "attack", "times_finalizacao", "Standard_G/Sh", false, false},
	{"possession", "attack", "times_padrao", "Poss", false, false},
	{"goals_against", "defense", "times_goleiros", "Performance_GA", true, true},
	{"shots_against", "defense", "times_finalizacao_adversarios", "Standard_Sh", true, true},
	{"shots_on_target_against", "defense", "times_finalizacao_adversarios", "Standard_SoT", true, true},
	{"save_pct", "defense", "times_goleiros", "Performance_Save%", false, false},
	{"clean_sheets", "defense", "times_goleiros", "Performance_CS", false, false},
	{"fouls", "discipline", "times_diversos", "Performance_Fls", true, true},
	{"yellow_cards", "discipline", "times_padrao", "Performance_CrdY", false, true},
	{"red_cards", "discipline", "times_padrao", "Performance_CrdR", false, true},
}

type csvTable struct {
	header map[string]int
	rows   [][]string
}

func (t *csvTable) get(row []string, column string) string {
	if i, ok := t.header[column]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

func (t *csvTable) num(row []string, column string) float64 {
	v, err := strconv.ParseFloat(strings.TrimPrefix(strings.ReplaceAll(t.get(row, column), ",", ""), "+"), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func readFBrefCSV(dir, name string, required ...string) (*csvTable, error) {
	f, err := os.Open(filepath.Join(dir, name+".csv"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(io.LimitReader(f, maxCSVBytes))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s.csv: %w", name, err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("%s.csv: sem linhas", name)
	}
	t := &csvTable{header: map[string]int{}, rows: records[1:]}
	for i, h := range records[0] {
		t.header[strings.TrimSpace(h)] = i
	}
	for _, col := range required {
		if _, ok := t.header[col]; !ok {
			return nil, fmt.Errorf("%s.csv: coluna %q ausente", name, col)
		}
	}
	return t, nil
}

// LoadFBref reads the CSV folder and computes per-club metrics, league averages and ranks.
func LoadFBref(dir string) (*FBrefSeason, error) {
	tables := map[string]*csvTable{}
	need := map[string][]string{"times_padrao": {"Squad", "Playing Time_MP"}, "classificacao_casa_fora": {"Squad"}, "classificacao": {"Squad", "Attendance"}, "jogos": {"Home", "Away", "Score", "Venue", "Date"}, "jogadores_padrao": {"Player", "Squad", "Performance_Gls", "Performance_Ast"}, "goleiros": {"Player", "Squad", "Playing Time_Min"}}
	for _, m := range fbrefMetrics {
		need[m.file] = append(need[m.file], "Squad", m.column)
	}
	for name, cols := range need {
		t, err := readFBrefCSV(dir, name, cols...)
		if err != nil {
			return nil, err
		}
		tables[name] = t
	}
	s := &FBrefSeason{clubs: map[string]*SeasonStats{}}
	matches := map[string]float64{}
	padrao := tables["times_padrao"]
	for _, row := range padrao.rows {
		squad := padrao.get(row, "Squad")
		if !validDataName(squad) {
			continue
		}
		matches[squad] = padrao.num(row, "Playing Time_MP")
		s.clubs[squad] = &SeasonStats{Source: "FBref (cópia local)", SourceURL: fbrefSourceURL, Team: squad, Matches: int(matches[squad]), Metrics: []SeasonMetric{}}
	}
	if len(s.clubs) < 2 {
		return nil, errors.New("FBref: times_padrao.csv sem clubes")
	}
	for _, m := range fbrefMetrics {
		t := tables[m.file]
		values := map[string]float64{}
		for _, row := range t.rows {
			squad := t.get(row, "Squad")
			if _, ok := s.clubs[squad]; !ok {
				continue
			}
			v := t.num(row, m.column)
			if m.perMatch && matches[squad] > 0 {
				v /= matches[squad]
			}
			values[squad] = math.Round(v*100) / 100
		}
		league := 0.0
		ordered := make([]float64, 0, len(values))
		for _, v := range values {
			league += v
			ordered = append(ordered, v)
		}
		if len(values) > 0 {
			league = math.Round(league/float64(len(values))*100) / 100
		}
		sort.Slice(ordered, func(i, j int) bool {
			if m.lowerIsBetter {
				return ordered[i] < ordered[j]
			}
			return ordered[i] > ordered[j]
		})
		for squad, v := range values {
			// Competition ranking: tied clubs share the best position.
			rank := sort.Search(len(ordered), func(i int) bool {
				if m.lowerIsBetter {
					return ordered[i] >= v
				}
				return ordered[i] <= v
			}) + 1
			s.clubs[squad].Metrics = append(s.clubs[squad].Metrics, SeasonMetric{Key: m.key, Group: m.group, Value: v, League: league, Rank: rank, Clubs: len(values), LowerIsBetter: m.lowerIsBetter})
		}
	}
	split := tables["classificacao_casa_fora"]
	for _, row := range split.rows {
		c := s.clubs[split.get(row, "Squad")]
		if c == nil {
			continue
		}
		record := func(side string) *SplitRecord {
			r := &SplitRecord{Played: int(split.num(row, side+"_MP")), Wins: int(split.num(row, side+"_W")), Draws: int(split.num(row, side+"_D")), Losses: int(split.num(row, side+"_L")), GoalsFor: int(split.num(row, side+"_GF")), GoalsAgainst: int(split.num(row, side+"_GA")), Points: int(split.num(row, side+"_Pts"))}
			if r.Played == 0 {
				return nil
			}
			r.PointsPerMatch = math.Round(float64(r.Points)/float64(r.Played)*100) / 100
			return r
		}
		c.Home, c.Away = record("Home"), record("Away")
	}
	table := tables["classificacao"]
	for _, row := range table.rows {
		if c := s.clubs[table.get(row, "Squad")]; c != nil {
			c.AverageAttendance = int(table.num(row, "Attendance"))
		}
	}
	// Stadium: the venue of most played home matches; as-of: the latest played match.
	games := tables["jogos"]
	venues := map[string]map[string]int{}
	for _, row := range games.rows {
		home, venue, date := games.get(row, "Home"), games.get(row, "Venue"), games.get(row, "Date")
		if games.get(row, "Score") == "" || s.clubs[home] == nil {
			continue
		}
		if date > s.asOf {
			s.asOf = date
		}
		if validDataName(venue) {
			if venues[home] == nil {
				venues[home] = map[string]int{}
			}
			venues[home][venue]++
		}
	}
	for squad, counts := range venues {
		s.clubs[squad].Stadium = mostCommon(counts)
	}
	players := tables["jogadores_padrao"]
	for _, row := range players.rows {
		c := s.clubs[players.get(row, "Squad")]
		name := players.get(row, "Player")
		if c == nil || !validDataName(name) {
			continue
		}
		for _, leader := range []struct {
			slot   **SeasonLeader
			column string
		}{{&c.TopScorer, "Performance_Gls"}, {&c.TopAssists, "Performance_Ast"}} {
			v := int(players.num(row, leader.column))
			if v > 0 && (*leader.slot == nil || v > (*leader.slot).Value) {
				*leader.slot = &SeasonLeader{Player: name, Value: v}
			}
		}
	}
	keepers := tables["goleiros"]
	minutes := map[string]float64{}
	for _, row := range keepers.rows {
		squad, name := keepers.get(row, "Squad"), keepers.get(row, "Player")
		c := s.clubs[squad]
		if c == nil || !validDataName(name) || keepers.num(row, "Playing Time_Min") <= minutes[squad] {
			continue
		}
		minutes[squad] = keepers.num(row, "Playing Time_Min")
		c.Goalkeeper = &SeasonKeeper{Player: name, Matches: int(keepers.num(row, "Playing Time_MP")), SavePct: keepers.num(row, "Performance_Save%"), CleanSheets: int(keepers.num(row, "Performance_CS"))}
	}
	for _, c := range s.clubs {
		c.AsOf = s.asOf
		order := map[string]int{}
		for i, m := range fbrefMetrics {
			order[m.key] = i
		}
		sort.Slice(c.Metrics, func(i, j int) bool { return order[c.Metrics[i].Key] < order[c.Metrics[j].Key] })
	}
	return s, nil
}

// AsOf is the date of the latest played match in the export.
func (s *FBrefSeason) AsOf() string { return s.asOf }

// Season returns a copy of the stats for an OpenFootball club, matching names through
// the shared Brazilian aliases ("CA Mineiro" ↔ "Atlético Mineiro").
func (s *FBrefSeason) Season(team Team) (*SeasonStats, bool) {
	key := func(name string) string {
		n := normalizeTeam(name)
		if alias, ok := brazilSearchAliases[n]; ok {
			return alias
		}
		return n
	}
	want := key(team.Name)
	var best *SeasonStats
	bestDiff := math.MaxInt
	for squad, c := range s.clubs {
		if diff, ok := nameDistance(want, key(squad)); ok && diff < bestDiff {
			best, bestDiff = c, diff
		}
	}
	if best == nil {
		return nil, false
	}
	out := *best
	out.Metrics = append([]SeasonMetric(nil), best.Metrics...)
	clone := func(p *SplitRecord) *SplitRecord {
		if p == nil {
			return nil
		}
		c := *p
		return &c
	}
	out.Home, out.Away = clone(best.Home), clone(best.Away)
	if best.TopScorer != nil {
		c := *best.TopScorer
		out.TopScorer = &c
	}
	if best.TopAssists != nil {
		c := *best.TopAssists
		out.TopAssists = &c
	}
	if best.Goalkeeper != nil {
		c := *best.Goalkeeper
		out.Goalkeeper = &c
	}
	return &out, true
}
