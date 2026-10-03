package football

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The dataset is GPL-2.0 (Kaggle: adaoduque/campeonato-brasileiro-de-futebol). It is
// downloaded at runtime from the author's GitHub mirror and never bundled here.
const historyURL = "https://raw.githubusercontent.com/adaoduque/Brasileirao_Dataset/master/campeonato-brasileiro-full.csv"
const maxHistoryBytes = 8 << 20

var ErrHistoryUnavailable = errors.New("histórico do Brasileirão indisponível no momento")

// HistorySource adds all-time Série A context to a snapshot.
type HistorySource interface {
	ClubHistory(ctx context.Context, team Team, opponents []Team) (*ClubHistory, error)
}

// HistoryProvider serves Série A results since 2003 from a fixed public CSV, cached
// for twelve hours with a failure cooldown, like the OpenFootball provider.
type HistoryProvider struct {
	url         string
	client      *http.Client
	gate        chan struct{}
	cached      *historyDataset
	ttl         time.Duration
	now         func() time.Time
	lastFailure error
	retryAt     time.Time
}

type historyMatch struct {
	id                           string
	season                       int
	date, home, away             string
	homeScore, awayScore         int
	arena                        string
	homeCoach, awayCoach         string
	homeFormation, awayFormation string
}

// Secondary files of the same dataset; each is optional.
type historyGoal struct {
	matchID, club, player, minute, kind string
}

type historyCard struct {
	season              int
	club, player, color string
}

type historyStats struct {
	shots, onTarget, possession, corners, fouls int
}

type historyDataset struct {
	matches     []historyMatch
	goals       map[string][]historyGoal // by match ID
	cards       []historyCard
	stats       map[string]map[string]historyStats // match ID → club → stats
	tables      map[int][]Standing
	missing     []string
	clubs       []string
	champions   map[string][]int
	first, last int
	fetchedAt   time.Time
}

func NewHistoryProvider() *HistoryProvider {
	return &HistoryProvider{
		url:    historyURL,
		client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		gate:   make(chan struct{}, 1), ttl: 12 * time.Hour, now: time.Now,
	}
}

func (h *HistoryProvider) load(ctx context.Context) (*historyDataset, error) {
	select {
	case h.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-h.gate }()
	now := h.now()
	if h.cached != nil && now.Sub(h.cached.fetchedAt) < h.ttl {
		return h.cached, nil
	}
	if h.lastFailure != nil && now.Before(h.retryAt) {
		return nil, h.lastFailure
	}
	data, err := h.fetch(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		h.lastFailure, h.retryAt = err, now.Add(time.Minute)
		return nil, err
	}
	h.cached, h.lastFailure = data, nil
	return data, nil
}

func (h *HistoryProvider) fetch(ctx context.Context) (*historyDataset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return nil, ErrHistoryUnavailable
	}
	req.Header.Set("User-Agent", "MatchMind/0.1 (Brasileirão history client)")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, ErrHistoryUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, ErrHistoryUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxHistoryBytes+1))
	if err != nil || len(body) > maxHistoryBytes {
		return nil, ErrHistoryUnavailable
	}
	data, err := parseHistory(string(body), h.now())
	if err != nil {
		return nil, err
	}
	// Goals, cards and statistics enrich the history; a failure only drops that part.
	base := h.url[:strings.LastIndex(h.url, "/")+1]
	for name, parse := range map[string]func(*historyDataset, string) error{
		"campeonato-brasileiro-gols.csv":              parseHistoryGoals,
		"campeonato-brasileiro-cartoes.csv":           parseHistoryCards,
		"campeonato-brasileiro-estatisticas-full.csv": parseHistoryStats,
	} {
		body, err := h.download(ctx, base+name)
		if err == nil {
			err = parse(data, body)
		}
		if err != nil {
			data.missing = append(data.missing, name)
		}
	}
	sort.Strings(data.missing)
	return data, nil
}

func (h *HistoryProvider) download(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", ErrHistoryUnavailable
	}
	req.Header.Set("User-Agent", "MatchMind/0.1 (Brasileirão history client)")
	res, err := h.client.Do(req)
	if err != nil {
		return "", ErrHistoryUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", ErrHistoryUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxHistoryBytes+1))
	if err != nil || len(body) > maxHistoryBytes {
		return "", ErrHistoryUnavailable
	}
	return string(body), nil
}

func parseHistory(body string, now time.Time) (*historyDataset, error) {
	reader := csv.NewReader(strings.NewReader(body))
	header, err := reader.Read()
	if err != nil {
		return nil, ErrHistoryUnavailable
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.TrimSpace(name)] = i
	}
	for _, name := range []string{"data", "mandante", "visitante", "mandante_Placar", "visitante_Placar"} {
		if _, ok := col[name]; !ok {
			return nil, ErrHistoryUnavailable
		}
	}
	data := &historyDataset{champions: map[string][]int{}, goals: map[string][]historyGoal{}, stats: map[string]map[string]historyStats{}, tables: map[int][]Standing{}, fetchedAt: now}
	clubs := map[string]bool{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) != len(header) || len(data.matches) >= 20000 {
			return nil, ErrHistoryUnavailable
		}
		day, err := time.Parse("02/01/2006", strings.TrimSpace(row[col["data"]]))
		home, away := strings.TrimSpace(row[col["mandante"]]), strings.TrimSpace(row[col["visitante"]])
		hs, errH := strconv.Atoi(strings.TrimSpace(row[col["mandante_Placar"]]))
		as, errA := strconv.Atoi(strings.TrimSpace(row[col["visitante_Placar"]]))
		if err != nil || errH != nil || errA != nil || hs < 0 || as < 0 || !validDataName(home) || !validDataName(away) || home == away {
			return nil, ErrHistoryUnavailable
		}
		// Seasons run within a calendar year, except 2020, which ended in February 2021.
		season := day.Year()
		if day.Month() <= time.February {
			season--
		}
		optional := func(name string) string {
			if i, ok := col[name]; ok {
				if v := strings.TrimSpace(row[i]); validDataName(v) {
					return v
				}
			}
			return ""
		}
		data.matches = append(data.matches, historyMatch{id: optional("ID"), season: season, date: day.Format("2006-01-02"), home: home, away: away, homeScore: hs, awayScore: as,
			arena: optional("arena"), homeCoach: optional("tecnico_mandante"), awayCoach: optional("tecnico_visitante"), homeFormation: optional("formacao_mandante"), awayFormation: optional("formacao_visitante")})
		clubs[home], clubs[away] = true, true
		if data.first == 0 || season < data.first {
			data.first = season
		}
		if season > data.last {
			data.last = season
		}
	}
	if len(data.matches) == 0 {
		return nil, ErrHistoryUnavailable
	}
	for club := range clubs {
		data.clubs = append(data.clubs, club)
	}
	sort.Strings(data.clubs)
	sort.SliceStable(data.matches, func(i, j int) bool { return data.matches[i].date > data.matches[j].date })
	data.computeChampions()
	return data, nil
}

// computeChampions takes the leader of each season's final table (points, wins,
// goal difference, goals scored). Results match the official list for 2003–2024.
func (d *historyDataset) computeChampions() {
	bySeason := map[int][]Match{}
	ids := map[string]Team{}
	for _, m := range d.matches {
		for _, name := range []string{m.home, m.away} {
			if _, ok := ids[name]; !ok {
				ids[name] = Team{ID: name, Name: name}
			}
		}
		bySeason[m.season] = append(bySeason[m.season], Match{HomeTeam: ids[m.home], AwayTeam: ids[m.away], HomeScore: m.homeScore, AwayScore: m.awayScore, Status: "finished"})
	}
	for season, matches := range bySeason {
		teams := map[string]Team{}
		for _, m := range matches {
			teams[m.HomeTeam.ID], teams[m.AwayTeam.ID] = m.HomeTeam, m.AwayTeam
		}
		list := []Team{}
		for _, t := range teams {
			list = append(list, t)
		}
		// Only complete double round-robins crown a champion (2016 misses one match).
		if n := len(list); n < 2 || float64(len(matches)) < 0.95*float64(n*(n-1)) {
			continue
		}
		table := CalculateStandings(list, matches)
		d.tables[season] = table
		if len(table) > 0 {
			d.champions[table[0].TeamID] = append(d.champions[table[0].TeamID], season)
		}
	}
	for club := range d.champions {
		sort.Ints(d.champions[club])
	}
}

// Dataset spellings that share no distinctive word with OpenFootball names.
var historyAliases = map[string]string{"ca mineiro": "atletico mg", "ca paranaense": "athletico pr"}

// datasetClub maps an OpenFootball name to the closest dataset spelling, or "".
// Token subsets are required; the candidate with the fewest extra words wins, so
// "Grêmio FBPA" maps to "Gremio" rather than "Gremio Prudente".
func (d *historyDataset) datasetClub(openName string) string {
	normalized := normalizeTeam(openName)
	name := openName
	if alias, ok := historyAliases[normalized]; ok {
		name = alias
	}
	a := clubTokens(name)
	best, bestDiff := "", 1<<30
	for _, club := range d.clubs {
		b := clubTokens(club)
		if !subset(a, b) && !subset(b, a) {
			continue
		}
		diff := 0
		for t := range a {
			if !b[t] {
				diff++
			}
		}
		for t := range b {
			if !a[t] {
				diff++
			}
		}
		if diff < bestDiff {
			best, bestDiff = club, diff
		}
	}
	return best
}

func (h *HistoryProvider) ClubHistory(ctx context.Context, team Team, opponents []Team) (*ClubHistory, error) {
	data, err := h.load(ctx)
	if err != nil {
		return nil, err
	}
	club := data.datasetClub(team.Name)
	result := &ClubHistory{Source: "adaoduque/Brasileirao_Dataset", SourceURL: h.url, FetchedAt: data.fetchedAt.UTC().Format(time.RFC3339), FirstSeason: data.first, LastSeason: data.last, DatasetName: club, Titles: []int{}, HeadToHead: []HeadToHead{}, Seasons: []HistorySeason{}, TopScorers: []HistoryScorer{}}
	if club == "" {
		return result, nil
	}
	result.Titles = append(result.Titles, data.champions[club]...)
	seasons := map[int]bool{}
	for _, m := range data.matches {
		if m.home == club || m.away == club {
			seasons[m.season] = true
			result.AllTime.add(m, club)
		}
	}
	result.SeasonsPlayed = len(seasons)
	for _, opponent := range opponents {
		other := data.datasetClub(opponent.Name)
		if other == "" || other == club {
			continue
		}
		h2h := HeadToHead{OpponentID: opponent.ID, OpponentName: opponent.Name}
		for _, m := range data.matches {
			if (m.home == club && m.away == other) || (m.home == other && m.away == club) {
				if len(h2h.Meetings) < 5 {
					h2h.Meetings = append(h2h.Meetings, data.historicMatch(m))
				}
				if h2h.LastMatch == nil {
					last := data.historicMatch(m)
					h2h.LastMatch = &last
				}
				h2h.Record.add(m, club)
			}
		}
		if h2h.Played > 0 {
			result.HeadToHead = append(result.HeadToHead, h2h)
		}
	}
	result.Seasons = data.seasons(club)
	result.TopScorers = data.topScorers(club)
	result.Discipline = data.discipline(club)
	for _, name := range data.missing {
		result.Unavailable = append(result.Unavailable, strings.TrimSuffix(strings.TrimPrefix(name, "campeonato-brasileiro-"), ".csv"))
	}
	sort.Slice(result.HeadToHead, func(i, j int) bool {
		if result.HeadToHead[i].Played != result.HeadToHead[j].Played {
			return result.HeadToHead[i].Played > result.HeadToHead[j].Played
		}
		return result.HeadToHead[i].OpponentName < result.HeadToHead[j].OpponentName
	})
	return result, nil
}

func (r *Record) add(m historyMatch, club string) {
	scored, conceded := m.homeScore, m.awayScore
	if m.away == club {
		scored, conceded = conceded, scored
	}
	r.Played++
	r.GoalsFor += scored
	r.GoalsAgainst += conceded
	switch {
	case scored > conceded:
		r.Wins++
	case scored == conceded:
		r.Draws++
	default:
		r.Losses++
	}
}

// readCSV returns rows as maps keyed by the trimmed header, requiring the given columns.
func readCSV(body string, required ...string) ([]map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(body))
	header, err := reader.Read()
	if err != nil {
		return nil, ErrHistoryUnavailable
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}
	have := map[string]bool{}
	for _, h := range header {
		have[h] = true
	}
	for _, name := range required {
		if !have[name] {
			return nil, ErrHistoryUnavailable
		}
	}
	rows := []map[string]string{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) != len(header) || len(rows) >= 60000 {
			return nil, ErrHistoryUnavailable
		}
		row := make(map[string]string, len(header))
		for i, v := range record {
			row[header[i]] = strings.TrimSpace(v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (d *historyDataset) matchByID() map[string]*historyMatch {
	byID := make(map[string]*historyMatch, len(d.matches))
	for i := range d.matches {
		if d.matches[i].id != "" {
			byID[d.matches[i].id] = &d.matches[i]
		}
	}
	return byID
}

func parseHistoryGoals(d *historyDataset, body string) error {
	rows, err := readCSV(body, "partida_id", "clube", "atleta", "minuto")
	if err != nil {
		return err
	}
	kinds := map[string]string{"": "normal", "Penalty": "penalty", "Gol Contra": "own_goal"}
	byID := d.matchByID()
	for _, r := range rows {
		m := byID[r["partida_id"]]
		kind, ok := kinds[r["tipo_de_gol"]]
		if m == nil || !ok || !validDataName(r["atleta"]) || (r["clube"] != m.home && r["clube"] != m.away) {
			continue
		}
		minute := r["minuto"]
		if len(minute) > 8 || minute == "" {
			minute = "?"
		}
		d.goals[m.id] = append(d.goals[m.id], historyGoal{matchID: m.id, club: r["clube"], player: r["atleta"], minute: minute, kind: kind})
	}
	return nil
}

func parseHistoryCards(d *historyDataset, body string) error {
	rows, err := readCSV(body, "partida_id", "clube", "cartao", "atleta")
	if err != nil {
		return err
	}
	colors := map[string]string{"Amarelo": "yellow", "Vermelho": "red"}
	byID := d.matchByID()
	for _, r := range rows {
		m := byID[r["partida_id"]]
		color, ok := colors[r["cartao"]]
		if m == nil || !ok || !validDataName(r["atleta"]) || (r["clube"] != m.home && r["clube"] != m.away) {
			continue
		}
		d.cards = append(d.cards, historyCard{season: m.season, club: r["clube"], player: r["atleta"], color: color})
	}
	return nil
}

// parseHistoryStats keeps only rows with real values: the feed fills missing
// seasons (before 2017 and 2024) with zeros and "None", which are not statistics.
func parseHistoryStats(d *historyDataset, body string) error {
	rows, err := readCSV(body, "partida_id", "clube", "chutes", "chutes_no_alvo", "posse_de_bola", "faltas", "escanteios")
	if err != nil {
		return err
	}
	byID := d.matchByID()
	number := func(v string) (int, bool) {
		n, err := strconv.Atoi(strings.TrimSuffix(v, "%"))
		return n, err == nil && n >= 0
	}
	for _, r := range rows {
		m := byID[r["partida_id"]]
		if m == nil || (r["clube"] != m.home && r["clube"] != m.away) {
			continue
		}
		shots, ok1 := number(r["chutes"])
		onTarget, ok2 := number(r["chutes_no_alvo"])
		possession, ok3 := number(r["posse_de_bola"])
		fouls, ok4 := number(r["faltas"])
		corners, ok5 := number(r["escanteios"])
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || shots == 0 || possession == 0 || possession > 100 {
			continue
		}
		if d.stats[m.id] == nil {
			d.stats[m.id] = map[string]historyStats{}
		}
		d.stats[m.id][r["clube"]] = historyStats{shots: shots, onTarget: onTarget, possession: possession, corners: corners, fouls: fouls}
	}
	return nil
}

func (d *historyDataset) historicMatch(m historyMatch) HistoricMatch {
	out := HistoricMatch{Season: m.season, Date: m.date, HomeTeam: m.home, AwayTeam: m.away, HomeScore: m.homeScore, AwayScore: m.awayScore, Stadium: m.arena}
	goals := d.goals[m.id]
	// Show scorers only when the goal list adds up to the final score.
	home := 0
	for _, g := range goals {
		if g.club == m.home {
			home++
		}
	}
	if len(goals) == m.homeScore+m.awayScore && home == m.homeScore {
		for _, g := range goals {
			side := "away"
			if g.club == m.home {
				side = "home"
			}
			out.Goals = append(out.Goals, Goal{Side: side, Minute: g.minute, Player: g.player, Kind: g.kind})
		}
		sort.SliceStable(out.Goals, func(i, j int) bool { return minuteValue(out.Goals[i].Minute) < minuteValue(out.Goals[j].Minute) })
	}
	return out
}

func mostCommon(values map[string]int) string {
	best, count := "", 0
	for v, n := range values {
		if n > count || (n == count && v < best) {
			best, count = v, n
		}
	}
	return best
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

// seasons summarizes each Série A season of the club, newest first.
func (d *historyDataset) seasons(club string) []HistorySeason {
	type acc struct {
		season                HistorySeason
		coaches               map[string]int
		formations, stadiums  map[string]int
		statsN, zeroOnTarget  int
		pos, shots, on, cr, f int
	}
	bySeason := map[int]*acc{}
	// Oldest first, so coaches appear in the order they took charge.
	for i := len(d.matches) - 1; i >= 0; i-- {
		m := d.matches[i]
		if m.home != club && m.away != club {
			continue
		}
		a := bySeason[m.season]
		if a == nil {
			a = &acc{season: HistorySeason{Season: m.season}, coaches: map[string]int{}, formations: map[string]int{}, stadiums: map[string]int{}}
			bySeason[m.season] = a
		}
		a.season.Record.add(m, club)
		coach, formation := m.awayCoach, m.awayFormation
		if m.home == club {
			coach, formation = m.homeCoach, m.homeFormation
			if m.arena != "" {
				a.stadiums[m.arena]++
			}
		}
		if coach != "" {
			a.coaches[coach]++
		}
		if formation != "" {
			a.formations[formation]++
		}
		if st, ok := d.stats[m.id][club]; ok {
			a.statsN++
			a.pos += st.possession
			a.shots += st.shots
			a.on += st.onTarget
			a.cr += st.corners
			a.f += st.fouls
			if st.onTarget == 0 {
				a.zeroOnTarget++
			}
		}
	}
	out := []HistorySeason{}
	for season, a := range bySeason {
		s := a.season
		s.Points = s.Wins*3 + s.Draws
		for _, row := range d.tables[season] {
			if row.TeamID == club {
				s.Position = row.Position
			}
		}
		// Coaches by matches in charge, so interim or assistant coaches come last.
		for coach, n := range a.coaches {
			s.Coaches = append(s.Coaches, HistoryCoach{Name: coach, Matches: n})
		}
		sort.Slice(s.Coaches, func(i, j int) bool {
			if s.Coaches[i].Matches != s.Coaches[j].Matches {
				return s.Coaches[i].Matches > s.Coaches[j].Matches
			}
			return s.Coaches[i].Name < s.Coaches[j].Name
		})
		s.Formation, s.Stadium = mostCommon(a.formations), mostCommon(a.stadiums)
		// Averages need most of the season's matches, and the source must have filled
		// shots on target (2016 rows are mostly zero, which is missing data, not reality).
		if a.statsN >= 10 && a.zeroOnTarget*10 < a.statsN*3 {
			n := float64(a.statsN)
			s.Averages = &SeasonAverages{Matches: a.statsN, Possession: round1(float64(a.pos) / n), Shots: round1(float64(a.shots) / n), ShotsOnTarget: round1(float64(a.on) / n), Corners: round1(float64(a.cr) / n), Fouls: round1(float64(a.f) / n)}
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Season > out[j].Season })
	return out
}

// topScorers lists the club's ten leading Série A scorers; own goals are excluded
// because they were scored by opponents.
func (d *historyDataset) topScorers(club string) []HistoryScorer {
	type acc struct {
		goals, penalties int
		first, last      int
	}
	byPlayer := map[string]*acc{}
	season := map[string]int{}
	for _, m := range d.matches {
		season[m.id] = m.season
	}
	for id, goals := range d.goals {
		for _, g := range goals {
			if g.club != club || g.kind == "own_goal" {
				continue
			}
			a := byPlayer[g.player]
			if a == nil {
				a = &acc{first: season[id], last: season[id]}
				byPlayer[g.player] = a
			}
			a.goals++
			if g.kind == "penalty" {
				a.penalties++
			}
			a.first, a.last = min(a.first, season[id]), max(a.last, season[id])
		}
	}
	out := []HistoryScorer{}
	for player, a := range byPlayer {
		seasons := strconv.Itoa(a.first)
		if a.last != a.first {
			seasons += "–" + strconv.Itoa(a.last)
		}
		out = append(out, HistoryScorer{Player: player, Goals: a.goals, Penalties: a.penalties, Seasons: seasons})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Goals != out[j].Goals {
			return out[i].Goals > out[j].Goals
		}
		return out[i].Player < out[j].Player
	})
	return out[:min(10, len(out))]
}

func (d *historyDataset) discipline(club string) *HistoryDiscipline {
	out := &HistoryDiscipline{MostBooked: []HistoryBooking{}}
	byPlayer := map[string]*HistoryBooking{}
	for _, c := range d.cards {
		if c.club != club {
			continue
		}
		if out.FirstSeason == 0 || c.season < out.FirstSeason {
			out.FirstSeason = c.season
		}
		out.LastSeason = max(out.LastSeason, c.season)
		b := byPlayer[c.player]
		if b == nil {
			b = &HistoryBooking{Player: c.player}
			byPlayer[c.player] = b
		}
		if c.color == "red" {
			out.Red++
			b.Red++
		} else {
			out.Yellow++
			b.Yellow++
		}
	}
	if out.Yellow+out.Red == 0 {
		return nil
	}
	for _, b := range byPlayer {
		out.MostBooked = append(out.MostBooked, *b)
	}
	sort.Slice(out.MostBooked, func(i, j int) bool {
		a, b := out.MostBooked[i], out.MostBooked[j]
		if a.Yellow+a.Red != b.Yellow+b.Red {
			return a.Yellow+a.Red > b.Yellow+b.Red
		}
		if a.Red != b.Red {
			return a.Red > b.Red
		}
		return a.Player < b.Player
	})
	out.MostBooked = out.MostBooked[:min(5, len(out.MostBooked))]
	return out
}
