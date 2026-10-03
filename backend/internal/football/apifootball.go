package football

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiFootballRoot = "https://v3.football.api-sports.io"

// API-Football league IDs for the OpenFootball leagues it can enrich.
var apiFootballLeagues = map[string]int{"br.1": 71}

var ErrStatsUnavailable = errors.New("estatísticas das partidas indisponíveis no momento")

// StatisticsSource adds per-match statistics to OpenFootball results. Implementations
// fill only matches they can identify with certainty and leave the rest nil.
type StatisticsSource interface {
	Enrich(ctx context.Context, selected Team, matches []Match) error
	Name() string
}

// APIFootballStats reads fixture statistics from the documented API-Football v3
// service. The key stays on the server; browser input never reaches the URL.
// The season fixture list is cached for six hours and finished-match statistics
// indefinitely, keeping usage well under the free plan's 100 requests per day.
type APIFootballStats struct {
	root, key, season string
	league            int
	client            *http.Client
	mu                sync.Mutex
	fixtures          []apiFixture
	fixturesAt        time.Time
	stats             map[int]*MatchStatistics
	missing           map[int]time.Time
	lastFailure       error
	retryAt           time.Time
	now               func() time.Time
}

type apiFixture struct {
	Fixture struct {
		ID     int    `json:"id"`
		Date   string `json:"date"`
		Status struct {
			Short string `json:"short"`
		} `json:"status"`
	} `json:"fixture"`
	Teams struct {
		Home apiTeam `json:"home"`
		Away apiTeam `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
}

type apiTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type apiTeamStatistics struct {
	Team       apiTeam `json:"team"`
	Statistics []struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"statistics"`
}

func NewAPIFootballStats(key, league, season string) (*APIFootballStats, error) {
	id, ok := apiFootballLeagues[league]
	year, err := strconv.Atoi(season)
	if !ok || err != nil || year < 2000 || year > 2100 {
		return nil, errors.New("API-Football: estatísticas disponíveis apenas para br.1 com temporada no formato 2026")
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 200 || strings.ContainsFunc(key, func(r rune) bool { return r <= ' ' || r > '~' }) {
		return nil, errors.New("API_FOOTBALL_KEY inválida")
	}
	return &APIFootballStats{
		root: apiFootballRoot, key: key, season: season, league: id,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		stats:  map[int]*MatchStatistics{}, missing: map[int]time.Time{}, now: time.Now,
	}, nil
}

func (a *APIFootballStats) Name() string { return "api-football" }

func (a *APIFootballStats) get(ctx context.Context, path string, query url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.root+path+"?"+query.Encode(), nil)
	if err != nil {
		return ErrStatsUnavailable
	}
	req.Header.Set("x-apisports-key", a.key)
	req.Header.Set("Accept", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return ErrStatsUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w (HTTP %d)", ErrStatsUnavailable, res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxDatasetBytes+1))
	if err != nil || len(body) > maxDatasetBytes {
		return ErrStatsUnavailable
	}
	var envelope struct {
		Errors   json.RawMessage `json:"errors"`
		Response json.RawMessage `json:"response"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return ErrStatsUnavailable
	}
	// Errors arrive with HTTP 200 as a non-empty object, e.g. plan or quota limits.
	var messages map[string]string
	if json.Unmarshal(envelope.Errors, &messages) == nil && len(messages) > 0 {
		parts := []string{}
		for _, message := range messages {
			if len(message) > 200 {
				message = message[:200]
			}
			parts = append(parts, message)
		}
		return fmt.Errorf("%w: %s", ErrStatsUnavailable, strings.Join(parts, "; "))
	}
	if json.Unmarshal(envelope.Response, out) != nil {
		return ErrStatsUnavailable
	}
	return nil
}

func (a *APIFootballStats) loadFixtures(ctx context.Context) ([]apiFixture, error) {
	now := a.now()
	if a.fixtures != nil && now.Sub(a.fixturesAt) < 6*time.Hour {
		return a.fixtures, nil
	}
	if a.lastFailure != nil && now.Before(a.retryAt) {
		return nil, a.lastFailure
	}
	var fixtures []apiFixture
	err := a.get(ctx, "/fixtures", url.Values{"league": {strconv.Itoa(a.league)}, "season": {a.season}}, &fixtures)
	if err != nil {
		if ctx.Err() == nil {
			a.lastFailure, a.retryAt = err, now.Add(5*time.Minute)
		}
		return nil, err
	}
	a.fixtures, a.fixturesAt, a.lastFailure = fixtures, now, nil
	return fixtures, nil
}

// Enrich sets Statistics on matches in place. It returns the first failure so the
// caller can report why statistics are missing; matched results are kept regardless.
func (a *APIFootballStats) Enrich(ctx context.Context, _ Team, matches []Match) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	fixtures, err := a.loadFixtures(ctx)
	if err != nil {
		return err
	}
	var firstErr error
	for i := range matches {
		fixture := findFixture(fixtures, matches[i])
		if fixture == nil {
			continue
		}
		stats, err := a.fixtureStatistics(ctx, fixture)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if stats != nil {
			copied := *stats
			matches[i].Statistics = &copied
		}
	}
	return firstErr
}

func (a *APIFootballStats) fixtureStatistics(ctx context.Context, fixture *apiFixture) (*MatchStatistics, error) {
	id := fixture.Fixture.ID
	if stats, ok := a.stats[id]; ok {
		return stats, nil
	}
	if at, ok := a.missing[id]; ok && a.now().Sub(at) < 6*time.Hour {
		return nil, nil
	}
	var teams []apiTeamStatistics
	if err := a.get(ctx, "/fixtures/statistics", url.Values{"fixture": {strconv.Itoa(id)}}, &teams); err != nil {
		return nil, err
	}
	stats := convertStatistics(fixture, teams)
	if stats == nil {
		a.missing[id] = a.now()
		return nil, nil
	}
	a.stats[id] = stats
	return stats, nil
}

func convertStatistics(fixture *apiFixture, teams []apiTeamStatistics) *MatchStatistics {
	var home, away map[string]int
	for _, team := range teams {
		values := map[string]int{}
		for _, stat := range team.Statistics {
			if v, ok := statValue(stat.Value); ok {
				values[stat.Type] = v
			}
		}
		switch team.Team.ID {
		case fixture.Teams.Home.ID:
			home = values
		case fixture.Teams.Away.ID:
			away = values
		}
	}
	if home == nil || away == nil {
		return nil
	}
	if _, ok := home["Total Shots"]; !ok {
		if _, ok := home["Ball Possession"]; !ok {
			return nil
		}
	}
	pair := func(name string) StatPair { return StatPair{Home: home[name], Away: away[name]} }
	return &MatchStatistics{Possession: pair("Ball Possession"), Shots: pair("Total Shots"), ShotsOnTarget: pair("Shots on Goal"), Corners: pair("Corner Kicks"), Fouls: pair("Fouls"), YellowCards: pair("Yellow Cards"), RedCards: pair("Red Cards")}
}

// statValue accepts integers, "55%" strings and null (absent, e.g. no cards).
func statValue(raw json.RawMessage) (int, bool) {
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n, n >= 0
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(s), "%"))
		return n, err == nil && n >= 0
	}
	return 0, false
}

// findFixture links an OpenFootball result to one finished API-Football fixture.
// Date (±1 day for time zones), final score and both club names must all agree.
func findFixture(fixtures []apiFixture, match Match) *apiFixture {
	day, err := time.Parse("2006-01-02", match.Date)
	if err != nil {
		return nil
	}
	for i := range fixtures {
		f := &fixtures[i]
		when, err := time.Parse(time.RFC3339, f.Fixture.Date)
		if err != nil || f.Goals.Home == nil || f.Goals.Away == nil {
			continue
		}
		if diff := when.Sub(day); diff < -24*time.Hour || diff > 48*time.Hour {
			continue
		}
		if *f.Goals.Home == match.HomeScore && *f.Goals.Away == match.AwayScore && sameClub(match.HomeTeam.Name, f.Teams.Home.Name) && sameClub(match.AwayTeam.Name, f.Teams.Away.Name) {
			return f
		}
	}
	return nil
}

var clubAliases = map[string]string{"ca mineiro": "atletico mg"}
var clubStopwords = map[string]bool{"fc": true, "ec": true, "sc": true, "cr": true, "ca": true, "se": true, "af": true, "fr": true, "fbc": true, "fbpa": true, "rb": true, "clube": true, "do": true, "da": true, "de": true}

func clubTokens(name string) map[string]bool {
	name = normalizeTeam(name)
	if alias, ok := clubAliases[name]; ok {
		name = alias
	}
	tokens := map[string]bool{}
	for _, word := range strings.Fields(name) {
		if !clubStopwords[word] {
			tokens[word] = true
		}
	}
	return tokens
}

func subset(a, b map[string]bool) bool {
	for token := range a {
		if !b[token] {
			return false
		}
	}
	return len(a) > 0
}

func sameClub(openName, apiName string) bool {
	a, b := clubTokens(openName), clubTokens(apiName)
	return subset(a, b) || subset(b, a)
}
