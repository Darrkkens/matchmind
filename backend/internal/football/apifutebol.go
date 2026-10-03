package football

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiFutebolRoot = "https://api.api-futebol.com.br/v1"

// API-Futebol championship IDs for the OpenFootball leagues it can enrich.
var apiFutebolChampionships = map[string]int{"br.1": 10}

var roundNumber = regexp.MustCompile(`[0-9]+`)

var errFutebolNotFound = errors.New("API-Futebol: recurso não encontrado")

// APIFutebolStats reads match statistics from API-Futebol (api-futebol.com.br).
// Every response goes through a ResponseCache: finished matches and closed rounds
// are stored without expiry, so each one costs a single request ever when the
// cache is PostgreSQL. Test keys (test_…) return sample data, labeled as such.
type APIFutebolStats struct {
	root, key    string
	championship int
	test         bool
	client       *http.Client
	cache        ResponseCache
	mu           sync.Mutex
	lastFailure  error
	retryAt      time.Time
	now          func() time.Time
}

type futebolTeam struct {
	ID   int    `json:"time_id"`
	Name string `json:"nome_popular"`
}

type futebolMatch struct {
	ID        int         `json:"partida_id"`
	Home      futebolTeam `json:"time_mandante"`
	Away      futebolTeam `json:"time_visitante"`
	HomeScore *int        `json:"placar_mandante"`
	AwayScore *int        `json:"placar_visitante"`
	Status    string      `json:"status"`
}

type futebolRound struct {
	Status  string         `json:"status"`
	Matches []futebolMatch `json:"partidas"`
}

type futebolSide struct {
	Possession string `json:"posse_de_bola"`
	Corners    *int   `json:"escanteios"`
	Fouls      *int   `json:"faltas"`
	Shots      struct {
		Total  *int `json:"total"`
		OnGoal *int `json:"no_gol"`
	} `json:"finalizacao"`
}

type futebolDetail struct {
	futebolMatch
	Statistics *struct {
		Home *futebolSide `json:"mandante"`
		Away *futebolSide `json:"visitante"`
	} `json:"estatisticas"`
	Cards map[string]struct {
		Home []json.RawMessage `json:"mandante"`
		Away []json.RawMessage `json:"visitante"`
	} `json:"cartoes"`
}

func NewAPIFutebolStats(key, league string, cache ResponseCache) (*APIFutebolStats, error) {
	id, ok := apiFutebolChampionships[league]
	if !ok {
		return nil, errors.New("API-Futebol: estatísticas disponíveis apenas para br.1")
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 200 || strings.ContainsFunc(key, func(r rune) bool { return r <= ' ' || r > '~' }) {
		return nil, errors.New("API_FUTEBOL_KEY inválida")
	}
	if cache == nil {
		cache = NewMemoryCache()
	}
	return &APIFutebolStats{
		root: apiFutebolRoot, key: key, championship: id, test: strings.HasPrefix(key, "test_"), cache: cache,
		client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		now:    time.Now,
	}, nil
}

func (a *APIFutebolStats) Name() string {
	if a.test {
		return "api-futebol-teste"
	}
	return "api-futebol"
}

// fetch returns the cached body or performs one upstream request. The caller
// decides the TTL from the decoded content and stores it with store().
func (a *APIFutebolStats) fetch(ctx context.Context, path string) ([]byte, bool, error) {
	key := "api-futebol:" + a.Name() + ":" + path
	if body, ok, err := a.cache.Get(ctx, key); err == nil && ok {
		if string(body) == "null" {
			return nil, true, errFutebolNotFound
		}
		return body, true, nil
	} else if err != nil {
		slog.Warn("response cache read failed", "error", err)
	}
	if a.lastFailure != nil && a.now().Before(a.retryAt) {
		return nil, false, a.lastFailure
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.root+path, nil)
	if err != nil {
		return nil, false, ErrStatsUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+a.key)
	req.Header.Set("Accept", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return nil, false, ErrStatsUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxDatasetBytes+1))
	if err != nil || len(body) > maxDatasetBytes {
		return nil, false, ErrStatsUnavailable
	}
	if res.StatusCode == http.StatusNotFound {
		// Negative entries also save quota; they expire in case the resource appears later.
		a.store(ctx, path, []byte("null"), 6*time.Hour)
		return nil, false, errFutebolNotFound
	}
	if res.StatusCode != http.StatusOK {
		err := fmt.Errorf("%w (API-Futebol HTTP %d)", ErrStatsUnavailable, res.StatusCode)
		var upstream struct {
			Message string `json:"message"`
		}
		// Plan and quota errors explain what to do (e.g. the championship is not in the plan).
		if json.Unmarshal(body, &upstream) == nil && validDataName(upstream.Message) {
			err = fmt.Errorf("%w (API-Futebol HTTP %d: %s)", ErrStatsUnavailable, res.StatusCode, upstream.Message)
		}
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 || res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			a.lastFailure, a.retryAt = err, a.now().Add(5*time.Minute)
		}
		return nil, false, err
	}
	if !json.Valid(body) {
		return nil, false, ErrStatsUnavailable
	}
	return body, false, nil
}

func (a *APIFutebolStats) store(ctx context.Context, path string, body []byte, ttl time.Duration) {
	if err := a.cache.Put(ctx, "api-futebol:"+a.Name()+":"+path, body, ttl); err != nil {
		slog.Warn("response cache write failed", "error", err)
	}
}

func (a *APIFutebolStats) round(ctx context.Context, n int) (*futebolRound, error) {
	path := fmt.Sprintf("/campeonatos/%d/rodadas/%d", a.championship, n)
	body, cached, err := a.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	var round futebolRound
	if json.Unmarshal(body, &round) != nil {
		return nil, ErrStatsUnavailable
	}
	if !cached {
		ttl := time.Hour
		if round.Status == "encerrada" {
			ttl = 0
		}
		a.store(ctx, path, body, ttl)
	}
	return &round, nil
}

func (a *APIFutebolStats) detail(ctx context.Context, id int) (*futebolDetail, error) {
	path := "/partidas/" + strconv.Itoa(id)
	body, cached, err := a.fetch(ctx, path)
	if err != nil {
		return nil, err
	}
	var detail futebolDetail
	if json.Unmarshal(body, &detail) != nil {
		return nil, ErrStatsUnavailable
	}
	if !cached {
		ttl := time.Hour
		if detail.Status == "finalizado" || a.test {
			ttl = 0
		}
		a.store(ctx, path, body, ttl)
	}
	return &detail, nil
}

// Enrich links each OpenFootball result to the API-Futebol match of the same round
// with the same clubs (and, outside test mode, the same final score).
func (a *APIFutebolStats) Enrich(ctx context.Context, _ Team, matches []Match) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var firstErr error
	for i := range matches {
		n, err := strconv.Atoi(roundNumber.FindString(matches[i].Round))
		if err != nil || n <= 0 || n > 60 {
			continue
		}
		round, err := a.round(ctx, n)
		if errors.Is(err, errFutebolNotFound) {
			continue
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var found *futebolMatch
		for j := range round.Matches {
			m := &round.Matches[j]
			if sameBrazilianClub(matches[i].HomeTeam.Name, m.Home.Name) && sameBrazilianClub(matches[i].AwayTeam.Name, m.Away.Name) {
				found = m
				break
			}
		}
		if found == nil || (!a.test && !sameScore(found, matches[i])) {
			continue
		}
		detail, err := a.detail(ctx, found.ID)
		if errors.Is(err, errFutebolNotFound) {
			continue
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !a.test && (detail.Status != "finalizado" || !sameScore(&detail.futebolMatch, matches[i])) {
			continue
		}
		matches[i].Statistics = detail.statistics()
	}
	if a.test {
		notice := "chave de teste da API-Futebol: as estatísticas são dados de exemplo, não as reais da partida"
		if firstErr != nil {
			notice += "; " + firstErr.Error()
		}
		return errors.New(notice)
	}
	return firstErr
}

func sameScore(m *futebolMatch, match Match) bool {
	return m.HomeScore != nil && m.AwayScore != nil && *m.HomeScore == match.HomeScore && *m.AwayScore == match.AwayScore
}

func (d *futebolDetail) statistics() *MatchStatistics {
	if d.Statistics == nil || d.Statistics.Home == nil || d.Statistics.Away == nil {
		return nil
	}
	h, w := d.Statistics.Home, d.Statistics.Away
	if h.Shots.Total == nil && h.Possession == "" {
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
	stats := &MatchStatistics{
		Possession:    StatPair{Home: percent(h.Possession), Away: percent(w.Possession)},
		Shots:         StatPair{Home: value(h.Shots.Total), Away: value(w.Shots.Total)},
		ShotsOnTarget: StatPair{Home: value(h.Shots.OnGoal), Away: value(w.Shots.OnGoal)},
		Corners:       StatPair{Home: value(h.Corners), Away: value(w.Corners)},
		Fouls:         StatPair{Home: value(h.Fouls), Away: value(w.Fouls)},
	}
	if yellow, ok := d.Cards["amarelo"]; ok {
		stats.YellowCards = StatPair{Home: len(yellow.Home), Away: len(yellow.Away)}
	}
	if red, ok := d.Cards["vermelho"]; ok {
		stats.RedCards = StatPair{Home: len(red.Home), Away: len(red.Away)}
	}
	return stats
}

// sameBrazilianClub compares names using the aliases shared by Brazilian sources
// (e.g. "CA Paranaense" ↔ "Athletico-PR"), so the two Atléticos never collide.
func sameBrazilianClub(openName, other string) bool {
	name := openName
	if alias, ok := historyAliases[normalizeTeam(openName)]; ok {
		name = alias
	}
	a, b := clubTokens(name), clubTokens(other)
	return subset(a, b) || subset(b, a)
}
