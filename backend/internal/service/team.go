package service

import (
	"context"
	"errors"
	"matchmind/internal/ai"
	"matchmind/internal/football"
	"strings"
	"sync"
	"time"
)

// recentMatches is how many finished matches a snapshot shows (and the form covers).
const recentMatches = 5

type TeamService struct {
	Provider football.FootballProvider
	Source   string
	Notice   string
	Analyst  MatchAnalyst // optional AI step of the simulation

	mu          sync.Mutex
	simulations map[string]cachedSimulation
}

func (s *TeamService) Resolve(ctx context.Context, input string) (*football.Snapshot, error) {
	ref, err := football.ParseReference(input)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	teams, err := s.Provider.SearchTeam(ctx, ref.Query)
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, football.ErrNotFound
	}
	for _, t := range teams {
		if strings.EqualFold(t.Name, ref.Query) {
			return s.Snapshot(ctx, t.ID)
		}
	}
	if len(teams) > 1 {
		return nil, football.ErrAmbiguous
	}
	return s.Snapshot(ctx, teams[0].ID)
}
func (s *TeamService) Snapshot(ctx context.Context, id string) (*football.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	if provider, ok := s.Provider.(football.SnapshotProvider); ok {
		return provider.GetSnapshot(ctx, id, recentMatches)
	}
	t, err := s.Provider.GetTeam(ctx, id)
	if err != nil {
		return nil, err
	}
	m, err := s.Provider.GetRecentMatches(ctx, id, recentMatches)
	if err != nil {
		return nil, err
	}
	p, err := s.Provider.GetSquad(ctx, id)
	if err != nil {
		return nil, err
	}
	tr, err := s.Provider.GetTrophies(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		m = []football.Match{}
	}
	if p == nil {
		p = []football.Player{}
	}
	if tr == nil {
		tr = []football.Trophy{}
	}
	return &football.Snapshot{Team: t, RecentMatches: m, Squad: p, Trophies: tr, RecentForm: football.CalculateForm(id, m), Standings: []football.Standing{}, DataSource: s.Source, DataNotice: s.Notice}, nil
}

// Standings returns the league table when the provider supports it.
func (s *TeamService) Standings(ctx context.Context) (*football.Table, error) {
	provider, ok := s.Provider.(football.StandingsProvider)
	if !ok {
		return nil, football.ErrStandingsUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	return provider.GetStandings(ctx)
}

// DefaultSimulationRuns is how many matches the next-fixture simulation plays by default.
const DefaultSimulationRuns = 50

// MatchAnalyst is the AI step of the simulation (see ai.Ollama.AnalyzeMatch).
type MatchAnalyst interface {
	AnalyzeMatch(ctx context.Context, facts string) (*football.SimulationAdjustment, error)
}

type cachedSimulation struct {
	sim *football.Simulation
	at  time.Time
}

const simulationCacheTTL = 15 * time.Minute

// Simulate runs the next-fixture model. With withAI, Gemma first reviews the inputs and its
// bounded adjustment becomes one more factor; the random draws always run here, so the
// percentages come from the simulation, not from the language model.
func (s *TeamService) Simulate(ctx context.Context, id string, runs int, withAI bool) (*football.Simulation, error) {
	if !validID.MatchString(id) {
		return nil, ErrValidation
	}
	if runs == 0 {
		runs = DefaultSimulationRuns
	}
	provider, ok := s.Provider.(football.Simulator)
	if !ok {
		return nil, football.ErrNoFixture
	}
	inputCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	in, notes, err := provider.SimulationInput(inputCtx, id)
	cancel()
	if err != nil {
		return nil, err
	}
	if _, err := football.Simulate(*in, runs); err != nil {
		return nil, err // invalid run count or data, before spending time on the AI
	}
	aiNote := ""
	switch {
	case !withAI:
		aiNote = "Simulação sem a análise da IA."
	case s.Analyst == nil:
		aiNote = "A análise da IA não está configurada."
	default:
		prelim, _ := football.Simulate(*in, 1)
		facts, err := ai.BuildAnalystFacts(prelim, in.ClubSeason, in.OpponentSeason)
		if err == nil {
			in.Analyst, err = s.Analyst.AnalyzeMatch(ctx, facts)
		}
		switch {
		case errors.Is(err, ai.ErrUngroundedReason):
			in.Analyst = nil
			aiNote = "A IA local respondeu, mas a justificativa dela não batia com os dados (números inexistentes ou argumento contrário ao ajuste); o ajuste não foi aplicado."
		case err != nil:
			aiNote = "A IA local não respondeu a tempo; a simulação seguiu sem o ajuste dela."
		}
	}
	sim, err := football.Simulate(*in, runs)
	if err != nil {
		return nil, err
	}
	if aiNote != "" {
		sim.Factors = append(sim.Factors, football.SimulationFactor{Key: "ai_analyst", Club: 1, Opponent: 1, Detail: aiNote})
	}
	sim.Notes = append(sim.Notes, notes...)
	if in.Analyst != nil {
		s.mu.Lock()
		if s.simulations == nil {
			s.simulations = map[string]cachedSimulation{}
		}
		s.simulations[id] = cachedSimulation{sim, time.Now()}
		s.mu.Unlock()
	}
	return sim, nil
}

// CachedSimulation returns the club's latest AI-reviewed simulation for its current next
// fixture, so the chat explains the numbers the user saw instead of recomputing them.
func (s *TeamService) CachedSimulation(id string, fixture *football.Fixture) *football.Simulation {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.simulations[id]
	if !ok || time.Since(c.at) > simulationCacheTTL || fixture == nil || c.sim.Fixture.Date != fixture.Date || c.sim.Fixture.HomeTeam.ID != fixture.HomeTeam.ID {
		return nil
	}
	return c.sim
}

// Lineups returns a match's team sheets when the provider supports them.
func (s *TeamService) Lineups(ctx context.Context, ref string) (*football.MatchLineups, error) {
	provider, ok := s.Provider.(football.LineupSource)
	if !ok {
		return nil, football.ErrLineupsUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return provider.Lineups(ctx, ref)
}

func (s *TeamService) Healthy(ctx context.Context) bool {
	_, err := s.Provider.SearchTeam(ctx, "Palmeiras")
	return err == nil
}
