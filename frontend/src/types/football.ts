export interface Team { id: string; name: string; short_name: string; country: string; logo_url: string; stadium: string; coach: string; founded_year: number }
export interface Player { id: string; name: string; position: string; number: number; appearances?: number; goals?: number; assists?: number; rating?: number }
export interface StatPair { home: number; away: number }
export interface MatchStatistics { possession: StatPair; shots: StatPair; shots_on_target: StatPair; corners: StatPair; fouls: StatPair; yellow_cards: StatPair; red_cards: StatPair }
export interface Goal { side: 'home' | 'away'; minute: string; player: string; assist?: string; kind: 'normal' | 'penalty' | 'own_goal' }
export interface Card { side: 'home' | 'away'; minute: string; player: string; color: 'yellow' | 'red' }
export interface Match { id: string; competition: string; date: string; round?: string; home_team: Team; away_team: Team; home_score: number; away_score: number; status: string; statistics: MatchStatistics | null; goals?: Goal[]; cards?: Card[]; venue?: string; referee?: string; lineup_ref?: string }
export interface Trophy { competition: string; count: number; seasons?: string[] }
export interface RecentForm { played: number; wins: number; draws: number; losses: number; goals_scored: number; goals_conceded: number; average_goals: number; points_percentage: number; sequence: string[] }
export interface DataMetadata { competition: string; season: string; source_url: string; fetched_at: string; latest_match_date: string; unavailable_fields: string[]; statistics_source?: string; statistics_notice?: string; history_notice?: string }
export interface MatchRecord { played: number; wins: number; draws: number; losses: number; goals_for: number; goals_against: number }
export interface HistoricMatch { season: number; date: string; home_team: string; away_team: string; home_score: number; away_score: number; stadium?: string; goals?: Goal[] }
export interface HeadToHead extends MatchRecord { opponent_id: string; opponent_name: string; last_match: HistoricMatch | null; meetings?: HistoricMatch[] }
export interface ClubHistory { source: string; source_url: string; fetched_at: string; first_season: number; last_season: number; dataset_name: string; seasons_played: number; titles: number[]; all_time: MatchRecord; head_to_head: HeadToHead[]; seasons?: HistorySeason[]; top_scorers?: HistoryScorer[]; discipline?: HistoryDiscipline; unavailable?: string[] }
export interface SeasonAverages { matches: number; possession: number; shots: number; shots_on_target: number; corners: number; fouls: number }
export interface HistorySeason extends MatchRecord { season: number; position?: number; points: number; coaches?: { name: string; matches: number }[]; formation?: string; stadium?: string; averages?: SeasonAverages }
export interface HistoryScorer { player: string; goals: number; penalties?: number; seasons: string }
export interface HistoryDiscipline { first_season: number; last_season: number; yellow: number; red: number; most_booked: { player: string; yellow: number; red: number }[] }
export interface Snapshot { team: Team; recent_matches: Match[]; trophies: Trophy[]; squad: Player[]; recent_form: RecentForm; standings: Standing[]; history?: ClubHistory; data_source: string; data_notice: string; data_metadata?: DataMetadata; next_match?: Fixture; season_stats?: SeasonStats; next_opponent_season?: SeasonStats; next_opponent_recent?: Match[]; next_match_availability?: MatchAvailability }
// A scheduled match with no result yet; time is the published local kick-off, when known.
export interface Fixture { competition: string; round?: string; date: string; time?: string; home_team: Team; away_team: Team }
// League-season totals from a dated local source; per-match metrics are already averaged.
export interface SeasonMetric { key: string; group: 'attack' | 'defense' | 'discipline'; value: number; league: number; rank: number; clubs: number; lower_is_better?: boolean }
export interface SplitRecord { played: number; wins: number; draws: number; losses: number; goals_for: number; goals_against: number; points: number; points_per_match: number }
export interface SeasonStats { source: string; source_url: string; as_of: string; team: string; matches: number; metrics: SeasonMetric[]; home?: SplitRecord; away?: SplitRecord; stadium?: string; average_attendance?: number; top_scorer?: { player: string; value: number }; top_assists?: { player: string; value: number }; goalkeeper?: { player: string; matches: number; save_pct: number; clean_sheets: number }; key_players?: KeyPlayer[]; finishers?: Finisher[] }
export interface KeyPlayer { player: string; position?: string; minutes: number; minutes_pct: number; plus_minus_90: number; on_off: number; points_per_match: number }
export interface Finisher { player: string; goals: number; shots: number; shots_on_target: number; accuracy_pct: number; goals_per_shot: number }
export interface Availability { suspended: { player: string; reason: string; minutes_pct?: number; on_off?: number }[]; at_risk: string[]; note?: string }
export interface MatchAvailability { club?: Availability; opponent?: Availability }
// Monte Carlo estimate for the next fixture from the selected club's side (percentages 0–100).
export interface SimulationFactor { key: string; club: number; opponent: number; detail: string; available: boolean; club_value?: number; opponent_value?: number }
export interface Simulation { runs: number; seed: number; fixture: Fixture; club_id: string; win_pct: number; draw_pct: number; loss_pct: number; expected_goals_club: number; expected_goals_opponent: number; top_scorelines: { club: number; opponent: number; percent: number }[]; factors: SimulationFactor[]; club_last5: string[]; opponent_last5: string[]; notes: string[] }
export interface ChatAnswer { answer: string; sources_used: string[] }
export interface Health { status: string; ollama: boolean; football_provider: boolean; data_source: string }
export interface Standing { position: number; team_id: string; team_name: string; logo_url: string; played: number; wins: number; draws: number; losses: number; goals_for: number; goals_against: number; goal_difference: number; points: number }
export interface LeagueTable { competition: string; season: string; source_url: string; fetched_at: string; standings: Standing[] }
export interface LineupPlayer { number: number; name: string; position: string; minutes: number; rating?: number; goals?: number; assists?: number; yellow?: number; red?: number }
export interface Lineup { team: string; formation?: string; coach?: string; starters: LineupPlayer[]; substitutes: LineupPlayer[] }
export interface MatchLineups { source: string; home: Lineup; away: Lineup }
