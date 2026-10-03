package football

import (
	"context"
	"errors"
)

var (
	ErrNotFound             = errors.New("clube não encontrado; confira o nome do clube e se ele está na liga coberta")
	ErrInvalidInput         = errors.New("digite o nome de um clube ou uma URL HTTP(S) com /football/team/{slug}/{id}")
	ErrAmbiguous            = errors.New("mais de um clube encontrado; digite o nome completo do clube")
	ErrProviderUnavailable  = errors.New("A OpenFootball está indisponível. Tente novamente em instantes.")
	ErrDatasetUnavailable   = errors.New("a liga/temporada configurada na OpenFootball está indisponível; confira OPENFOOTBALL_LEAGUE e OPENFOOTBALL_SEASON")
	ErrStandingsUnavailable = errors.New("a tabela do campeonato não está disponível nesta fonte de dados")
	ErrProviderData         = errors.New("A OpenFootball retornou dados inválidos. Tente novamente mais tarde.")
)

// Providers use their own stable IDs, never a URL's external reference.
// Return completed matches newest first; missing statistics must be nil.
type FootballProvider interface {
	SearchTeam(context.Context, string) ([]Team, error)
	GetTeam(context.Context, string) (*Team, error)
	GetRecentMatches(context.Context, string, int) ([]Match, error)
	GetSquad(context.Context, string) ([]Player, error)
	GetTrophies(context.Context, string) ([]Trophy, error)
}

// SnapshotProvider lets remote providers build all sections from one dataset revision.
// The five FootballProvider methods remain the minimum adapter contract.
type SnapshotProvider interface {
	GetSnapshot(context.Context, string, int) (*Snapshot, error)
}

// SquadSource lists a club's players (optional; empty when unknown).
type SquadSource interface {
	Squad(ctx context.Context, team Team) ([]Player, error)
}

// LineupSource returns a match's team sheets from an opaque reference that the
// same source placed on Match.LineupRef.
type LineupSource interface {
	Lineups(ctx context.Context, ref string) (*MatchLineups, error)
}

var ErrLineupsUnavailable = errors.New("escalações indisponíveis para esta partida")

// StandingsProvider exposes the league table of the configured competition.
type StandingsProvider interface {
	GetStandings(context.Context) (*Table, error)
}
