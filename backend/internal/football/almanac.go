package football

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const almanacRoot = "https://api.almanacstats.com"

// AlmanacStats uses the free, keyless AlmanacStats API (https://almanacstats.com/api-docs)
// in a limited, on-demand way: only the opened club is requested, at most one request
// per second as the provider asks, and every response goes through a ResponseCache
// (finished matches never expire). Its terms restrict use to personal, non-commercial
// purposes; see the README before enabling it elsewhere.
type AlmanacStats struct {
	root        string
	client      *http.Client
	cache       ResponseCache
	mu          sync.Mutex
	lastRequest time.Time
	interval    time.Duration
	lastFailure error
	retryAt     time.Time
	now         func() time.Time
}

type almanacTeam struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Score *int   `json:"score"`
}

type almanacListMatch struct {
	Home almanacTeam `json:"home"`
	Away almanacTeam `json:"away"`
}

type almanacClubMatch struct {
	MID       int         `json:"mid"`
	Status    string      `json:"status"`
	Round     string      `json:"round"`
	Side      string      `json:"side"`
	HomeScore *int        `json:"home_score"`
	AwayScore *int        `json:"away_score"`
	League    string      `json:"league"`
	Opponent  almanacTeam `json:"opponent"`
}

type almanacPlayer struct {
	Slug     string          `json:"slug"`
	Name     string          `json:"name"`
	Position string          `json:"position"`
	Apps     int             `json:"apps"`
	Goals    int             `json:"goals"`
	Assists  int             `json:"assists"`
	Rating   json.RawMessage `json:"rating"`
}

type almanacProfile struct {
	Club *struct {
		League struct {
			Slug string `json:"slug"`
		} `json:"league"`
		Matches []almanacClubMatch `json:"matches"`
		Squad   []almanacPlayer    `json:"squad"`
	} `json:"club"`
}

type almanacSide struct {
	Possession string `json:"possession"`
	Shots      *int   `json:"shots"`
	ShotsOn    *int   `json:"shots_on"`
	Corners    *int   `json:"corners"`
	Fouls      *int   `json:"fouls"`
	Yellow     *int   `json:"yellow"`
	Red        *int   `json:"red"`
}

type almanacEvent struct {
	Side   string          `json:"side"`
	Type   string          `json:"type"`
	Detail string          `json:"detail"`
	Minute json.RawMessage `json:"minute"`
	Player string          `json:"player"`
	Assist string          `json:"assist"`
}

type almanacDetail struct {
	Status    string `json:"status"`
	Home      almanacTeam
	Away      almanacTeam
	Events    []almanacEvent
	Venue     string
	Referee   string
	Lineups   map[string]almanacSheet
	Players   map[string][]almanacMatchPlayer
	TeamStats *struct {
		Home *almanacSide `json:"home"`
		Away *almanacSide `json:"away"`
	} `json:"team_stats"`
}

func (d *almanacDetail) UnmarshalJSON(raw []byte) error {
	var wire struct {
		Status string         `json:"status"`
		Home   almanacTeam    `json:"home"`
		Away   almanacTeam    `json:"away"`
		Events []almanacEvent `json:"events"`
		Venue  *struct {
			Name string `json:"name"`
			City string `json:"city"`
		} `json:"venue"`
		Referee    *string                 `json:"referee"`
		TeamStats  json.RawMessage         `json:"team_stats"`
		Lineups    map[string]almanacSheet `json:"lineups"`
		LineupHome []almanacMatchPlayer    `json:"lineup_home"`
		LineupAway []almanacMatchPlayer    `json:"lineup_away"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	d.Status, d.Home, d.Away, d.Events = wire.Status, wire.Home, wire.Away, wire.Events
	d.Lineups = wire.Lineups
	d.Players = map[string][]almanacMatchPlayer{"home": wire.LineupHome, "away": wire.LineupAway}
	if v := wire.Venue; v != nil && validDataName(v.Name) {
		d.Venue = strings.TrimSpace(v.Name)
		if validDataName(v.City) {
			d.Venue += " (" + strings.TrimSpace(v.City) + ")"
		}
	}
	if r := wire.Referee; r != nil && validDataName(*r) {
		// The feed appends the referee's country ("Name, Brazil").
		name, _, _ := strings.Cut(*r, ",")
		d.Referee = strings.TrimSpace(name)
	}
	if len(wire.TeamStats) > 0 && string(wire.TeamStats) != "null" {
		return json.Unmarshal(wire.TeamStats, &d.TeamStats)
	}
	return nil
}

var errAlmanacNotFound = errors.New("AlmanacStats: recurso não encontrado")

func NewAlmanacStats(cache ResponseCache) *AlmanacStats {
	if cache == nil {
		cache = NewMemoryCache()
	}
	return &AlmanacStats{
		root:   almanacRoot,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		cache:  cache, interval: 1100 * time.Millisecond, now: time.Now,
	}
}

func (a *AlmanacStats) Name() string { return "almanacstats" }

// get returns a cached body or performs one rate-limited request. ttl picks the
// cache lifetime from the decoded body; 0 means it never expires.
func (a *AlmanacStats) get(ctx context.Context, path string, out any, ttl func([]byte) time.Duration) error {
	key := "almanacstats:" + path
	if body, ok, err := a.cache.Get(ctx, key); err == nil && ok {
		if string(body) == "null" {
			return errAlmanacNotFound
		}
		return json.Unmarshal(body, out)
	} else if err != nil {
		slog.Warn("response cache read failed", "error", err)
	}
	if a.lastFailure != nil && a.now().Before(a.retryAt) {
		return a.lastFailure
	}
	if wait := a.interval - a.now().Sub(a.lastRequest); wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	a.lastRequest = a.now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.root+path, nil)
	if err != nil {
		return ErrStatsUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MatchMind/0.1 (open-source, non-commercial; cached; https://github.com)")
	res, err := a.client.Do(req)
	if err != nil {
		return ErrStatsUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxDatasetBytes+1))
	if err != nil || len(body) > maxDatasetBytes {
		return ErrStatsUnavailable
	}
	if res.StatusCode == http.StatusNotFound {
		a.store(ctx, key, []byte("null"), 6*time.Hour)
		return errAlmanacNotFound
	}
	if res.StatusCode != http.StatusOK {
		err := fmt.Errorf("%w (AlmanacStats HTTP %d)", ErrStatsUnavailable, res.StatusCode)
		a.lastFailure, a.retryAt = err, a.now().Add(5*time.Minute)
		return err
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(body, &envelope) != nil || len(envelope.Data) == 0 || json.Unmarshal(envelope.Data, out) != nil {
		return ErrStatsUnavailable
	}
	a.store(ctx, key, envelope.Data, ttl(envelope.Data))
	return nil
}

func (a *AlmanacStats) store(ctx context.Context, key string, body []byte, ttl time.Duration) {
	if err := a.cache.Put(ctx, key, body, ttl); err != nil {
		slog.Warn("response cache write failed", "error", err)
	}
}

func fixedTTL(d time.Duration) func([]byte) time.Duration {
	return func([]byte) time.Duration { return d }
}

// teamSlug finds the AlmanacStats slug for an OpenFootball br.1 club using the
// league's recent results (cached for a day).
func (a *AlmanacStats) teamSlug(ctx context.Context, name string) (string, error) {
	var list []almanacListMatch
	if err := a.get(ctx, "/v1/matches?"+url.Values{"scope": {"finished"}, "league": {"serie-a-brazil"}}.Encode(), &list, fixedTTL(24*time.Hour)); err != nil {
		return "", err
	}
	teams := map[string]string{}
	for _, m := range list {
		teams[m.Home.Name], teams[m.Away.Name] = m.Home.Slug, m.Away.Slug
	}
	best, bestDiff := "", 1<<30
	for teamName, slug := range teams {
		if diff, ok := nameDistance(name, teamName); ok && diff < bestDiff && validSlug(slug) {
			best, bestDiff = slug, diff
		}
	}
	return best, nil
}

func validSlug(slug string) bool {
	if slug == "" || len(slug) > 100 {
		return false
	}
	for _, r := range slug {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// nameDistance compares club names by distinctive words without source-specific
// aliases ("CA Mineiro" ⊂ "Atlético Mineiro"); fewer extra words is a closer match.
func nameDistance(a, b string) (int, bool) {
	ta, tb := plainTokens(a), plainTokens(b)
	if !subset(ta, tb) && !subset(tb, ta) {
		return 0, false
	}
	diff := 0
	for t := range ta {
		if !tb[t] {
			diff++
		}
	}
	for t := range tb {
		if !ta[t] {
			diff++
		}
	}
	return diff, true
}

func plainTokens(name string) map[string]bool {
	tokens := map[string]bool{}
	for _, word := range strings.Fields(normalizeTeam(name)) {
		if !clubStopwords[word] {
			tokens[word] = true
		}
	}
	return tokens
}

func (a *AlmanacStats) profile(ctx context.Context, team Team) (*almanacProfile, error) {
	slug, err := a.teamSlug(ctx, team.Name)
	if err != nil {
		return nil, err
	}
	if slug == "" {
		return nil, errAlmanacNotFound
	}
	var profile almanacProfile
	if err := a.get(ctx, "/v1/team/"+slug+"/profile", &profile, fixedTTL(6*time.Hour)); err != nil {
		return nil, err
	}
	if profile.Club == nil {
		return nil, errAlmanacNotFound
	}
	return &profile, nil
}

// Enrich links OpenFootball results to AlmanacStats matches of the selected club by
// round, opponent and final score, then reads team statistics from the match page.
func (a *AlmanacStats) Enrich(ctx context.Context, selected Team, matches []Match) error {
	if len(matches) == 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	profile, err := a.profile(ctx, selected)
	if errors.Is(err, errAlmanacNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var firstErr error
	for i := range matches {
		m := &matches[i]
		home := m.HomeTeam.ID == selected.ID
		opponent := m.AwayTeam.Name
		if !home {
			opponent = m.HomeTeam.Name
		}
		round := roundNumber.FindString(m.Round)
		var found *almanacClubMatch
		for j := range profile.Club.Matches {
			c := &profile.Club.Matches[j]
			if c.Status != "FT" || c.HomeScore == nil || c.AwayScore == nil || roundNumber.FindString(c.Round) != round || (c.Side == "home") != home {
				continue
			}
			if _, ok := nameDistance(opponent, c.Opponent.Name); ok && *c.HomeScore == m.HomeScore && *c.AwayScore == m.AwayScore {
				found = c
				break
			}
		}
		if found == nil || found.MID <= 0 {
			continue
		}
		var detail almanacDetail
		err := a.get(ctx, "/v1/match/"+strconv.Itoa(found.MID), &detail, func(body []byte) time.Duration {
			var probe struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(body, &probe) == nil && probe.Status == "FT" {
				return 0
			}
			return time.Hour
		})
		if errors.Is(err, errAlmanacNotFound) {
			continue
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		// The match page must agree with OpenFootball on both clubs and the score.
		_, homeOK := nameDistance(m.HomeTeam.Name, detail.Home.Name)
		_, awayOK := nameDistance(m.AwayTeam.Name, detail.Away.Name)
		if !homeOK || !awayOK || detail.Home.Score == nil || detail.Away.Score == nil || *detail.Home.Score != m.HomeScore || *detail.Away.Score != m.AwayScore {
			continue
		}
		m.Statistics = detail.statistics()
		m.Goals = detail.goals(m.HomeScore, m.AwayScore)
		m.Cards = detail.cards()
		m.Venue, m.Referee = detail.Venue, detail.Referee
		if detail.hasLineups() {
			m.LineupRef = strconv.Itoa(found.MID)
		}
	}
	return firstErr
}

// goals lists scorers in minute order, or nil when the events do not add up to the
// final score (incomplete feeds must not show a partial list as if it were complete).
func (d *almanacDetail) goals(homeScore, awayScore int) []Goal {
	kinds := map[string]string{"Normal Goal": "normal", "Penalty": "penalty", "Own Goal": "own_goal"}
	goals := []Goal{}
	count := map[string]int{}
	for _, e := range d.Events {
		kind, ok := kinds[e.Detail]
		if e.Type != "Goal" || !ok || (e.Side != "home" && e.Side != "away") || !validDataName(e.Player) {
			continue
		}
		minute := eventMinute(e.Minute)
		assist := strings.TrimSpace(e.Assist)
		if !validDataName(assist) || kind == "own_goal" {
			assist = ""
		}
		goals = append(goals, Goal{Side: e.Side, Minute: minute, Player: cleanName(e.Player), Assist: cleanName(assist), Kind: kind})
		count[e.Side]++
	}
	if count["home"] != homeScore || count["away"] != awayScore || len(goals) == 0 {
		return nil
	}
	sort.SliceStable(goals, func(i, j int) bool { return minuteValue(goals[i].Minute) < minuteValue(goals[j].Minute) })
	return goals
}

// cards lists bookings in minute order. VAR reviews are separate events and are
// ignored; a second yellow arrives as its own red card event.
func (d *almanacDetail) cards() []Card {
	colors := map[string]string{"Yellow Card": "yellow", "Red Card": "red"}
	cards := []Card{}
	for _, e := range d.Events {
		color, ok := colors[e.Detail]
		if e.Type != "Card" || !ok || (e.Side != "home" && e.Side != "away") || !validDataName(e.Player) {
			continue
		}
		cards = append(cards, Card{Side: e.Side, Minute: eventMinute(e.Minute), Player: cleanName(e.Player), Color: color})
	}
	if len(cards) == 0 {
		return nil
	}
	sort.SliceStable(cards, func(i, j int) bool { return minuteValue(cards[i].Minute) < minuteValue(cards[j].Minute) })
	return cards
}

func eventMinute(raw json.RawMessage) string {
	minute := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if minute == "" || minute == "null" || len(minute) > 8 || strings.ContainsAny(minute, "\\{}[]") {
		return "?"
	}
	return minute
}

// minuteValue orders "45+2" after "45" and before "46".
func minuteValue(minute string) float64 {
	base, extra, _ := strings.Cut(minute, "+")
	b, err := strconv.Atoi(base)
	if err != nil {
		return 1e9
	}
	e, _ := strconv.Atoi(extra)
	return float64(b) + float64(e)/100
}

func (d *almanacDetail) statistics() *MatchStatistics {
	if d.TeamStats == nil || d.TeamStats.Home == nil || d.TeamStats.Away == nil {
		return nil
	}
	h, w := d.TeamStats.Home, d.TeamStats.Away
	if h.Shots == nil && h.Possession == "" {
		return nil
	}
	value := func(p *int) int {
		if p == nil || *p < 0 {
			return 0
		}
		return *p
	}
	percent := func(s string) int {
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "%"))
		if err != nil || n < 0 || n > 100 {
			return 0
		}
		return n
	}
	return &MatchStatistics{
		Possession:    StatPair{Home: percent(h.Possession), Away: percent(w.Possession)},
		Shots:         StatPair{Home: value(h.Shots), Away: value(w.Shots)},
		ShotsOnTarget: StatPair{Home: value(h.ShotsOn), Away: value(w.ShotsOn)},
		Corners:       StatPair{Home: value(h.Corners), Away: value(w.Corners)},
		Fouls:         StatPair{Home: value(h.Fouls), Away: value(w.Fouls)},
		YellowCards:   StatPair{Home: value(h.Yellow), Away: value(w.Yellow)},
		RedCards:      StatPair{Home: value(h.Red), Away: value(w.Red)},
	}
}

var almanacPositions = map[string]string{"Goalkeeper": "Goleiro", "Defender": "Defensor", "Midfield": "Meio-campo", "Midfielder": "Meio-campo", "Attack": "Atacante", "Attacker": "Atacante", "Forward": "Atacante"}

// Squad returns the club's squad with the totals AlmanacStats reports per player.
func (a *AlmanacStats) Squad(ctx context.Context, team Team) ([]Player, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	profile, err := a.profile(ctx, team)
	if errors.Is(err, errAlmanacNotFound) {
		return []Player{}, nil
	}
	if err != nil {
		return nil, err
	}
	players := []Player{}
	for _, p := range profile.Club.Squad {
		if !validDataName(p.Name) || len(players) >= 80 {
			continue
		}
		position := almanacPositions[p.Position]
		if position == "" {
			position = "Não informada"
		}
		rating := 0.0
		var text string
		if json.Unmarshal(p.Rating, &text) == nil {
			rating, _ = strconv.ParseFloat(text, 64)
		} else {
			_ = json.Unmarshal(p.Rating, &rating)
		}
		players = append(players, Player{ID: "al-" + p.Slug, Name: cleanName(p.Name), Position: position, Appearances: p.Apps, Goals: p.Goals, Assists: p.Assists, Rating: rating})
	}
	return players, nil
}

type almanacSheetPlayer struct {
	Pos    string `json:"pos"`
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type almanacSheet struct {
	Formation string               `json:"formation"`
	Coach     string               `json:"coach"`
	Start     []almanacSheetPlayer `json:"start"`
	Subs      []almanacSheetPlayer `json:"subs"`
}

type almanacMatchPlayer struct {
	Name     string          `json:"name"`
	Position string          `json:"position"`
	Minutes  int             `json:"minutes"`
	Rating   json.RawMessage `json:"rating"`
	Goals    int             `json:"goals"`
	Assists  int             `json:"assists"`
	Yellow   int             `json:"yellow"`
	Red      int             `json:"red"`
}

func (d *almanacDetail) hasLineups() bool {
	home, away := d.Lineups["home"], d.Lineups["away"]
	return len(home.Start) > 0 && len(away.Start) > 0
}

var sheetPositions = map[string]string{"G": "Goleiro", "D": "Defensor", "M": "Meio-campo", "F": "Atacante"}

func parseRating(raw json.RawMessage) float64 {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		v, _ := strconv.ParseFloat(text, 64)
		return v
	}
	var v float64
	_ = json.Unmarshal(raw, &v)
	return v
}

// lineup merges the team sheet with per-player match numbers by name.
func (d *almanacDetail) lineup(side, team string) Lineup {
	sheet := d.Lineups[side]
	out := Lineup{Team: team, Starters: []LineupPlayer{}, Substitutes: []LineupPlayer{}}
	if validDataName(sheet.Formation) {
		out.Formation = sheet.Formation
	}
	if validDataName(sheet.Coach) {
		out.Coach = strings.TrimSpace(sheet.Coach)
	}
	stats := d.Players[side]
	convert := func(p almanacSheetPlayer) (LineupPlayer, bool) {
		if !validDataName(p.Name) || p.Number < 0 || p.Number > 999 {
			return LineupPlayer{}, false
		}
		player := LineupPlayer{Number: p.Number, Name: cleanName(p.Name), Position: sheetPositions[p.Pos]}
		best := -1
		bestDiff := 1 << 30
		for i, s := range stats {
			if diff, ok := playerDistance(player.Name, cleanName(s.Name)); ok && diff < bestDiff {
				best, bestDiff = i, diff
			}
		}
		if best >= 0 {
			s := stats[best]
			player.Minutes, player.Rating, player.Goals, player.Assists, player.Yellow, player.Red = s.Minutes, parseRating(s.Rating), s.Goals, s.Assists, s.Yellow, s.Red
			if player.Position == "" {
				player.Position = almanacPositions[s.Position]
			}
		}
		return player, true
	}
	for _, p := range sheet.Start {
		if player, ok := convert(p); ok && len(out.Starters) < 15 {
			out.Starters = append(out.Starters, player)
		}
	}
	for _, p := range sheet.Subs {
		if player, ok := convert(p); ok && len(out.Substitutes) < 25 {
			out.Substitutes = append(out.Substitutes, player)
		}
	}
	return out
}

// Lineups returns both team sheets for a match reference set by Enrich. The
// reference is only a numeric match ID; the host is fixed.
func (a *AlmanacStats) Lineups(ctx context.Context, ref string) (*MatchLineups, error) {
	mid, err := strconv.Atoi(ref)
	if err != nil || mid <= 0 || len(ref) > 10 {
		return nil, ErrLineupsUnavailable
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var detail almanacDetail
	err = a.get(ctx, "/v1/match/"+strconv.Itoa(mid), &detail, func(body []byte) time.Duration {
		var probe struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(body, &probe) == nil && probe.Status == "FT" {
			return 0
		}
		return time.Hour
	})
	if errors.Is(err, errAlmanacNotFound) || (err == nil && !detail.hasLineups()) {
		return nil, ErrLineupsUnavailable
	}
	if err != nil {
		return nil, err
	}
	return &MatchLineups{Source: a.Name(), Home: detail.lineup("home", detail.Home.Name), Away: detail.lineup("away", detail.Away.Name)}, nil
}

// playerDistance matches team-sheet names to full names: "G. Gómez" ↔ "Gustavo Gómez",
// "Osorio Luis" ↔ "Luis Osorio". Every word of the shorter name must appear in the
// longer one, where a one-letter word matches as an initial. Lower is closer.
func playerDistance(a, b string) (int, bool) {
	ta, tb := strings.Fields(normalizeTeam(a)), strings.Fields(normalizeTeam(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0, false
	}
	if len(ta) > len(tb) {
		ta, tb = tb, ta
	}
	used := make([]bool, len(tb))
	full := 0
	for _, word := range ta {
		found := false
		for j, other := range tb {
			if used[j] {
				continue
			}
			if word == other || (len(word) == 1 && strings.HasPrefix(other, word)) {
				used[j], found = true, true
				if len(word) > 1 {
					full++
				}
				break
			}
		}
		if !found {
			return 0, false
		}
	}
	// At least one complete word (usually the surname) must match.
	if full == 0 {
		return 0, false
	}
	return len(tb) - len(ta), true
}

// cleanName repairs two artifacts seen in the feed: HTML entities ("Sant&apos;Anna")
// and UTF-8 text decoded as Latin-1 ("JoÃ£o" for "João").
func cleanName(name string) string {
	name = strings.TrimSpace(html.UnescapeString(name))
	if !strings.ContainsAny(name, "ÃÂ") {
		return name
	}
	raw := make([]byte, 0, len(name))
	for _, r := range name {
		if r > 0xFF {
			return name
		}
		raw = append(raw, byte(r))
	}
	if fixed := string(raw); utf8.ValidString(fixed) {
		return fixed
	}
	return name
}
