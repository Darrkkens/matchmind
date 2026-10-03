// Package footballtest provides a fictional in-memory provider for tests only.
package footballtest

import (
	"context"
	"strings"

	"matchmind/internal/football"
)

// All sporting details are deliberately fictional and never served by the application.
type Provider struct{ teams []football.Team }

func NewProvider() *Provider {
	return &Provider{teams: []football.Team{
		{ID: "demo-palmeiras", Name: "Palmeiras", ShortName: "PAL", LogoURL: "/crests/palmeiras.svg", Country: "Brazil", Stadium: "Jardim Arena (demo)", Coach: "Rafael Costa (demo)", FoundedYear: 1914},
		{ID: "demo-aurora", Name: "Aurora FC", ShortName: "AUR", Country: "Brazil", Stadium: "Aurora Park (demo)", Coach: "Marina Alves (demo)", FoundedYear: 1982},
		{ID: "demo-harbor", Name: "Harbor United", ShortName: "HBR", Country: "England", Stadium: "Harbor Ground (demo)", Coach: "Alex Morgan (demo)", FoundedYear: 1926},
	}}
}
func (p *Provider) SearchTeam(ctx context.Context, q string) ([]football.Team, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q = strings.ToLower(strings.TrimSpace(q))
	result := []football.Team{}
	if q == "" {
		return result, nil
	}
	for _, t := range p.teams {
		if strings.Contains(strings.ToLower(t.Name), q) || strings.EqualFold(t.ShortName, q) {
			result = append(result, t)
		}
	}
	return result, nil
}
func (p *Provider) GetTeam(ctx context.Context, id string) (*football.Team, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, t := range p.teams {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, football.ErrNotFound
}
func (p *Provider) GetRecentMatches(ctx context.Context, id string, limit int) ([]football.Match, error) {
	t, err := p.GetTeam(ctx, id)
	if err != nil {
		return nil, err
	}
	opponents := []football.Team{}
	for _, o := range p.teams {
		if o.ID != id {
			opponents = append(opponents, o)
		}
	}
	// Separate exhibition series for each club; IDs intentionally include the club ID.
	matches := []football.Match{
		{ID: id + "-1", Competition: "Demo Invitational", Date: "2026-09-27T18:00:00Z", HomeTeam: *t, AwayTeam: opponents[0], HomeScore: 2, AwayScore: 0, Status: "finished", Statistics: demoStats(61, 18, 7, 9, 2)},
		{ID: id + "-2", Competition: "Demo Invitational", Date: "2026-09-20T18:00:00Z", HomeTeam: opponents[1], AwayTeam: *t, HomeScore: 1, AwayScore: 1, Status: "finished", Statistics: demoStats(46, 10, 4, 12, 5)},
		{ID: id + "-3", Competition: "Demo Cup", Date: "2026-09-13T18:00:00Z", HomeTeam: *t, AwayTeam: opponents[1], HomeScore: 2, AwayScore: 1, Status: "finished", Statistics: demoStats(54, 15, 6, 11, 4)},
	}
	if limit < 0 {
		limit = 0
	}
	if limit < len(matches) {
		matches = matches[:limit]
	}
	return matches, nil
}
func demoStats(pos, shots, target, awayShots, awayTarget int) *football.MatchStatistics {
	return &football.MatchStatistics{Possession: football.StatPair{Home: pos, Away: 100 - pos}, Shots: football.StatPair{Home: shots, Away: awayShots}, ShotsOnTarget: football.StatPair{Home: target, Away: awayTarget}, Corners: football.StatPair{Home: 6, Away: 3}, Fouls: football.StatPair{Home: 11, Away: 13}, YellowCards: football.StatPair{Home: 2, Away: 3}, RedCards: football.StatPair{Home: 0, Away: 0}}
}
func (p *Provider) GetSquad(ctx context.Context, id string) ([]football.Player, error) {
	if _, err := p.GetTeam(ctx, id); err != nil {
		return nil, err
	}
	return []football.Player{{ID: id + "-p1", Name: "Lucas Vale", Position: "Goalkeeper", Number: 1}, {ID: id + "-p2", Name: "Theo Martins", Position: "Defender", Number: 4}, {ID: id + "-p3", Name: "Gabriel Luz", Position: "Defender", Number: 6}, {ID: id + "-p4", Name: "Davi Rocha", Position: "Midfielder", Number: 8}, {ID: id + "-p5", Name: "Nico Santos", Position: "Midfielder", Number: 10}, {ID: id + "-p6", Name: "Mateo Silva", Position: "Forward", Number: 9}}, nil
}
func (p *Provider) GetTrophies(ctx context.Context, id string) ([]football.Trophy, error) {
	if _, err := p.GetTeam(ctx, id); err != nil {
		return nil, err
	}
	return []football.Trophy{{Competition: "Demo Invitational", Count: 2, Seasons: []string{"2023 (demo)", "2025 (demo)"}}, {Competition: "Demo Cup", Count: 1, Seasons: []string{"2024 (demo)"}}}, nil
}
