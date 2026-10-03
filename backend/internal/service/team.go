package service

import (
	"context"
	"matchmind/internal/football"
	"strings"
	"time"
)

type TeamService struct {
	Provider football.FootballProvider
	Source   string
	Notice   string
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
		return provider.GetSnapshot(ctx, id, 3)
	}
	t, err := s.Provider.GetTeam(ctx, id)
	if err != nil {
		return nil, err
	}
	m, err := s.Provider.GetRecentMatches(ctx, id, 3)
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
