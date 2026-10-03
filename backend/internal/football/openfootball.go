package football

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const OpenFootballNotice = "Resultados reais do conjunto de dados comunitário OpenFootball, não um feed ao vivo. Apenas a liga e a temporada configuradas são cobertas; estas podem não ser as partidas mais recentes do clube em todas as competições. Técnico, estádio, ano de fundação, elenco e títulos de outras competições não são fornecidos; estatísticas das partidas só aparecem quando uma API de estatísticas (API-Futebol ou API-Football) está configurada."
const openFootballRoot = "https://raw.githubusercontent.com/openfootball/football.json/master/"
const maxDatasetBytes = 4 << 20

var seasonPattern = regexp.MustCompile(`^[0-9]{4}(-[0-9]{2})?$`)
var leagueCountries = map[string]string{"br.1": "Brazil", "en.1": "England", "de.1": "Germany", "es.1": "Spain", "it.1": "Italy", "fr.1": "France"}

// OpenFootballProvider fetches the documented public JSON service, never user URLs.
// One validated dataset is cached in memory for five minutes. Failed refreshes are
// surfaced instead of substituting fictional or silently expired records.
type OpenFootballProvider struct {
	url, league, season, country string
	client                       *http.Client
	gate                         chan struct{}
	cached                       *openDataset
	ttl                          time.Duration
	now                          func() time.Time
	lastFailure                  error
	retryAt                      time.Time
	stats                        StatisticsSource
	history                      HistorySource
	squad                        SquadSource
}

type openDataset struct {
	competition string
	teams       []Team
	matches     []Match
	standings   []Standing
	fetchedAt   time.Time
}
type openDocument struct {
	Name    string `json:"name"`
	Matches []struct {
		Round string    `json:"round"`
		Date  string    `json:"date"`
		Team1 string    `json:"team1"`
		Team2 string    `json:"team2"`
		Score openScore `json:"score"`
	} `json:"matches"`
}

type openScore struct {
	FT []*int `json:"ft"`
}

// The published feed contains both {"ft":[h,a]} and compact [h,a] scores,
// including 0-0 draws. An absent/null score is different from either format.
func (s *openScore) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '[' {
		return json.Unmarshal(raw, &s.FT)
	}
	type scoreObject openScore
	return json.Unmarshal(raw, (*scoreObject)(s))
}

func NewOpenFootballProvider(league, season string) (*OpenFootballProvider, error) {
	country, ok := leagueCountries[league]
	if !ok || !seasonPattern.MatchString(season) {
		return nil, errors.New("configuração OpenFootball inválida: use br.1/en.1/de.1/es.1/it.1/fr.1 e uma temporada como 2026 ou 2026-27")
	}
	return &OpenFootballProvider{
		url: openFootballRoot + season + "/" + league + ".json", league: league, season: season, country: country,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		gate:   make(chan struct{}, 1), ttl: 5 * time.Minute, now: time.Now,
	}, nil
}

// SetStatistics enables optional per-match statistics; scores still come only from OpenFootball.
func (p *OpenFootballProvider) SetStatistics(source StatisticsSource) { p.stats = source }

// SetSquad enables squad lists from an external source.
func (p *OpenFootballProvider) SetSquad(source SquadSource) { p.squad = source }

// SetHistory enables all-time Série A context (titles, head-to-head) for snapshots.
func (p *OpenFootballProvider) SetHistory(source HistorySource) { p.history = source }

func (p *OpenFootballProvider) load(ctx context.Context) (*openDataset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case p.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-p.gate }()
	now := p.now()
	if p.cached != nil && now.Sub(p.cached.fetchedAt) < p.ttl {
		return p.cached, nil
	}
	if p.lastFailure != nil && now.Before(p.retryAt) {
		return nil, p.lastFailure
	}
	data, err := p.fetch(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		p.lastFailure = err
		p.retryAt = now.Add(15 * time.Second)
		return nil, err
	}
	p.cached = data
	p.lastFailure = nil
	return data, nil
}

func (p *OpenFootballProvider) fetch(ctx context.Context) (*openDataset, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MatchMind/0.1 (OpenFootball public dataset client)")
	res, err := p.client.Do(req)
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, ErrDatasetUnavailable
	}
	if res.StatusCode != http.StatusOK {
		return nil, ErrProviderUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxDatasetBytes+1))
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	if len(body) > maxDatasetBytes {
		return nil, ErrProviderData
	}
	var doc openDocument
	if json.Unmarshal(body, &doc) != nil || !validDataName(doc.Name) || len(doc.Matches) == 0 || len(doc.Matches) > 10000 {
		return nil, ErrProviderData
	}
	now := p.now()
	data := &openDataset{competition: doc.Name, teams: []Team{}, matches: []Match{}, fetchedAt: now}
	teams := map[string]Team{}
	seen := map[string]bool{}
	for _, row := range doc.Matches {
		if !validDataName(row.Team1) || !validDataName(row.Team2) || len(row.Round) > 120 || normalizeTeam(row.Team1) == normalizeTeam(row.Team2) {
			return nil, ErrProviderData
		}
		home, away := p.team(row.Team1), p.team(row.Team2)
		teams[home.ID] = home
		teams[away.ID] = away
		// Missing full-time scores are unplayed/unknown, never 0-0.
		if len(row.Score.FT) == 0 {
			continue
		}
		if len(row.Score.FT) != 2 || row.Score.FT[0] == nil || row.Score.FT[1] == nil || *row.Score.FT[0] < 0 || *row.Score.FT[1] < 0 {
			return nil, ErrProviderData
		}
		if _, err := time.Parse("2006-01-02", row.Date); err != nil {
			return nil, ErrProviderData
		}
		if row.Date > now.UTC().Format("2006-01-02") {
			continue
		}
		id := stableID("of-match", p.league+"/"+p.season+"/"+row.Round+"/"+row.Date+"/"+home.ID+"/"+away.ID)
		if seen[id] {
			return nil, ErrProviderData
		}
		seen[id] = true
		data.matches = append(data.matches, Match{ID: id, Competition: doc.Name, Round: row.Round, Date: row.Date, HomeTeam: home, AwayTeam: away, HomeScore: *row.Score.FT[0], AwayScore: *row.Score.FT[1], Status: "finished", Statistics: nil})
	}
	for _, team := range teams {
		data.teams = append(data.teams, team)
	}
	sort.Slice(data.teams, func(i, j int) bool { return data.teams[i].Name < data.teams[j].Name })
	data.standings = CalculateStandings(data.teams, data.matches)
	sort.Slice(data.matches, func(i, j int) bool {
		if data.matches[i].Date == data.matches[j].Date {
			return data.matches[i].ID < data.matches[j].ID
		}
		return data.matches[i].Date > data.matches[j].Date
	})
	return data, nil
}

func validDataName(name string) bool {
	return strings.TrimSpace(name) != "" && utf8.ValidString(name) && utf8.RuneCountInString(name) <= 120 && !strings.ContainsFunc(name, unicode.IsControl)
}
func normalizeTeam(name string) string {
	name = strings.ToLower(name)
	name = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e", "í", "i", "ï", "i", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ú", "u", "ü", "u", "ç", "c").Replace(name)
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, name)
	return strings.Join(strings.Fields(name), " ")
}
func stableID(prefix, value string) string {
	return fmt.Sprintf("%s-%x", prefix, sha256.Sum256([]byte(value)))[:len(prefix)+1+24]
}
func (p *OpenFootballProvider) team(name string) Team {
	// Display source names faithfully; artwork comes from a separate curated catalog.
	initials := []rune{}
	for _, word := range strings.Fields(name) {
		r, _ := utf8.DecodeRuneInString(word)
		if len(initials) < 3 {
			initials = append(initials, unicode.ToUpper(r))
		}
	}
	return Team{ID: stableID("of-team", p.league+"/"+normalizeTeam(name)), Name: name, ShortName: string(initials), Country: p.country, LogoURL: clubCrest(p.league, name)}
}

// Common Brazilian names and nicknames for OpenFootball's br.1 spellings.
var brazilSearchAliases = map[string]string{
	"atletico mineiro": "ca mineiro", "atletico mg": "ca mineiro", "galo": "ca mineiro",
	"athletico": "ca paranaense", "athletico paranaense": "ca paranaense", "athletico pr": "ca paranaense", "atletico paranaense": "ca paranaense", "atletico pr": "ca paranaense", "furacao": "ca paranaense",
	"timao": "corinthians", "verdao": "palmeiras", "mengao": "flamengo", "fogao": "botafogo", "peixe": "santos",
	"colorado": "internacional", "inter": "internacional", "spfc": "sao paulo", "chape": "chapecoense", "red bull bragantino": "bragantino",
}

func (p *OpenFootballProvider) SearchTeam(ctx context.Context, query string) ([]Team, error) {
	data, err := p.load(ctx)
	if err != nil {
		return nil, err
	}
	query = normalizeTeam(query)
	if alias, ok := brazilSearchAliases[query]; ok && p.league == "br.1" {
		query = alias
	}
	result := []Team{}
	if query == "" {
		return result, nil
	}
	for _, team := range data.teams {
		if strings.Contains(normalizeTeam(team.Name), query) {
			result = append(result, team)
		}
	}
	return result, nil
}
func findOpenTeam(data *openDataset, id string) (*Team, error) {
	for _, team := range data.teams {
		if team.ID == id {
			return &team, nil
		}
	}
	return nil, ErrNotFound
}
func (p *OpenFootballProvider) GetTeam(ctx context.Context, id string) (*Team, error) {
	data, err := p.load(ctx)
	if err != nil {
		return nil, err
	}
	return findOpenTeam(data, id)
}
func openMatches(data *openDataset, id string, limit int) []Match {
	result := []Match{}
	if limit <= 0 {
		return result
	}
	for _, match := range data.matches {
		if match.HomeTeam.ID == id || match.AwayTeam.ID == id {
			result = append(result, match)
			if len(result) == limit {
				break
			}
		}
	}
	return result
}
func (p *OpenFootballProvider) GetRecentMatches(ctx context.Context, id string, limit int) ([]Match, error) {
	data, err := p.load(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = findOpenTeam(data, id); err != nil {
		return nil, err
	}
	return openMatches(data, id, limit), nil
}
func (p *OpenFootballProvider) GetSquad(ctx context.Context, id string) ([]Player, error) {
	if _, err := p.GetTeam(ctx, id); err != nil {
		return nil, err
	}
	return []Player{}, nil
}
func (p *OpenFootballProvider) GetTrophies(ctx context.Context, id string) ([]Trophy, error) {
	if _, err := p.GetTeam(ctx, id); err != nil {
		return nil, err
	}
	return []Trophy{}, nil
}
func (p *OpenFootballProvider) GetSnapshot(ctx context.Context, id string, limit int) (*Snapshot, error) {
	data, err := p.load(ctx)
	if err != nil {
		return nil, err
	}
	team, err := findOpenTeam(data, id)
	if err != nil {
		return nil, err
	}
	matches := openMatches(data, id, limit)
	unavailable := []string{"coach", "stadium", "founded_year", "squad", "trophies", "match_statistics"}
	statsSource, statsNotice := "", ""
	if p.stats != nil && len(matches) > 0 {
		// Statistics are best effort: a slow or exhausted quota must not block results.
		statsCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		err := p.stats.Enrich(statsCtx, *team, matches)
		cancel()
		complete := true
		for _, m := range matches {
			if m.Statistics != nil {
				statsSource = p.stats.Name()
			} else {
				complete = false
			}
		}
		if complete {
			unavailable = unavailable[:len(unavailable)-1]
		}
		if err != nil {
			statsNotice = err.Error()
		} else if !complete {
			statsNotice = "algumas partidas não têm estatísticas na API-Football"
		}
	}
	squad := []Player{}
	if p.squad != nil {
		squadCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		players, err := p.squad.Squad(squadCtx, *team)
		cancel()
		if err == nil && len(players) > 0 {
			squad = players
			for i, field := range unavailable {
				if field == "squad" {
					unavailable = append(unavailable[:i:i], unavailable[i+1:]...)
					break
				}
			}
		}
	}
	trophies := []Trophy{}
	var history *ClubHistory
	historyNotice := ""
	if p.history != nil {
		opponents := []Team{}
		for _, t := range data.teams {
			if t.ID != id {
				opponents = append(opponents, t)
			}
		}
		historyCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		h, err := p.history.ClubHistory(historyCtx, *team, opponents)
		cancel()
		switch {
		case err != nil:
			historyNotice = err.Error()
		default:
			history = h
			for i, field := range unavailable {
				if field == "trophies" {
					unavailable[i] = "trophies_other_competitions"
				}
			}
			if len(h.Titles) > 0 {
				seasons := []string{}
				for _, year := range h.Titles {
					seasons = append(seasons, strconv.Itoa(year))
				}
				trophies = append(trophies, Trophy{Competition: fmt.Sprintf("Brasileirão Série A (%d–%d)", h.FirstSeason, h.LastSeason), Count: len(h.Titles), Seasons: seasons})
			}
		}
	}
	latest := ""
	if len(matches) > 0 {
		latest = matches[0].Date
	}
	return &Snapshot{Team: team, RecentMatches: matches, Squad: squad, Trophies: trophies, History: history, RecentForm: CalculateForm(id, matches), Standings: append([]Standing{}, data.standings...), DataSource: "openfootball", DataNotice: OpenFootballNotice,
		DataMetadata: &DataMetadata{Competition: data.competition, Season: p.season, SourceURL: p.url, FetchedAt: data.fetchedAt.UTC().Format(time.RFC3339), LatestMatchDate: latest, UnavailableFields: unavailable, StatisticsSource: statsSource, StatisticsNotice: statsNotice, HistoryNotice: historyNotice},
	}, nil
}

// GetStandings returns the league table computed from the cached dataset revision.
func (p *OpenFootballProvider) GetStandings(ctx context.Context) (*Table, error) {
	data, err := p.load(ctx)
	if err != nil {
		return nil, err
	}
	return &Table{Competition: data.competition, Season: p.season, SourceURL: p.url, FetchedAt: data.fetchedAt.UTC().Format(time.RFC3339), Standings: append([]Standing{}, data.standings...)}, nil
}

// Lineups delegates to the statistics source when it can serve team sheets.
func (p *OpenFootballProvider) Lineups(ctx context.Context, ref string) (*MatchLineups, error) {
	source, ok := p.stats.(LineupSource)
	if !ok {
		return nil, ErrLineupsUnavailable
	}
	return source.Lineups(ctx, ref)
}
