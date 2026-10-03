package football

type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ShortName   string `json:"short_name"`
	Country     string `json:"country"`
	LogoURL     string `json:"logo_url"`
	Stadium     string `json:"stadium"`
	Coach       string `json:"coach"`
	FoundedYear int    `json:"founded_year"`
}

type Player struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Position    string  `json:"position"`
	Number      int     `json:"number"`
	Appearances int     `json:"appearances,omitempty"`
	Goals       int     `json:"goals,omitempty"`
	Assists     int     `json:"assists,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
}

// Each pair is ordered home/away. A nil Statistics means unavailable, not zero.
type StatPair struct {
	Home int `json:"home"`
	Away int `json:"away"`
}
type MatchStatistics struct {
	Possession    StatPair `json:"possession"`
	Shots         StatPair `json:"shots"`
	ShotsOnTarget StatPair `json:"shots_on_target"`
	Corners       StatPair `json:"corners"`
	Fouls         StatPair `json:"fouls"`
	YellowCards   StatPair `json:"yellow_cards"`
	RedCards      StatPair `json:"red_cards"`
}
type Match struct {
	ID          string           `json:"id"`
	Competition string           `json:"competition"`
	Round       string           `json:"round,omitempty"`
	Date        string           `json:"date"`
	HomeTeam    Team             `json:"home_team"`
	AwayTeam    Team             `json:"away_team"`
	HomeScore   int              `json:"home_score"`
	AwayScore   int              `json:"away_score"`
	Status      string           `json:"status"`
	Statistics  *MatchStatistics `json:"statistics"`
	Goals       []Goal           `json:"goals,omitempty"`
	Cards       []Card           `json:"cards,omitempty"`
	Venue       string           `json:"venue,omitempty"`
	Referee     string           `json:"referee,omitempty"`
	// LineupRef is an opaque reference for GET /api/lineups/{ref} when lineups exist.
	LineupRef string `json:"lineup_ref,omitempty"`
}

// Card belongs to the side of the booked player.
type Card struct {
	Side   string `json:"side"` // home or away
	Minute string `json:"minute"`
	Player string `json:"player"`
	Color  string `json:"color"` // yellow or red
}

// Goal is credited to the side that benefits (an own goal counts for the opponent).
type Goal struct {
	Side   string `json:"side"` // home or away
	Minute string `json:"minute"`
	Player string `json:"player"`
	Assist string `json:"assist,omitempty"`
	Kind   string `json:"kind"` // normal, penalty or own_goal
}
type Trophy struct {
	Competition string   `json:"competition"`
	Count       int      `json:"count"`
	Seasons     []string `json:"seasons,omitempty"`
}
type RecentForm struct {
	Played           int      `json:"played"`
	Wins             int      `json:"wins"`
	Draws            int      `json:"draws"`
	Losses           int      `json:"losses"`
	GoalsScored      int      `json:"goals_scored"`
	GoalsConceded    int      `json:"goals_conceded"`
	AverageGoals     float64  `json:"average_goals"`
	PointsPercentage float64  `json:"points_percentage"`
	Sequence         []string `json:"sequence"`
}
type Snapshot struct {
	Team          *Team         `json:"team"`
	RecentMatches []Match       `json:"recent_matches"`
	Trophies      []Trophy      `json:"trophies"`
	Squad         []Player      `json:"squad"`
	RecentForm    RecentForm    `json:"recent_form"`
	Standings     []Standing    `json:"standings"`
	History       *ClubHistory  `json:"history,omitempty"`
	DataSource    string        `json:"data_source"`
	DataNotice    string        `json:"data_notice"`
	DataMetadata  *DataMetadata `json:"data_metadata,omitempty"`
	NextMatch     *Fixture      `json:"next_match,omitempty"`
	// The next opponent's latest finished league matches (scores only), newest first.
	NextOpponentRecent []Match `json:"next_opponent_recent,omitempty"`
	// Season totals from an optional local source; the next opponent's only with a next match.
	SeasonStats        *SeasonStats `json:"season_stats,omitempty"`
	NextOpponentSeason *SeasonStats `json:"next_opponent_season,omitempty"`
}

// SeasonStats are league-season totals for one club as of AsOf (a dated snapshot).
// Per-match metrics are already divided by Matches; League is the average of all clubs
// and Rank is 1 for the best value (lowest when LowerIsBetter).
type SeasonStats struct {
	Source            string         `json:"source"`
	SourceURL         string         `json:"source_url"`
	AsOf              string         `json:"as_of"`
	Team              string         `json:"team"`
	Matches           int            `json:"matches"`
	Metrics           []SeasonMetric `json:"metrics"`
	Home              *SplitRecord   `json:"home,omitempty"`
	Away              *SplitRecord   `json:"away,omitempty"`
	Stadium           string         `json:"stadium,omitempty"`
	AverageAttendance int            `json:"average_attendance,omitempty"`
	TopScorer         *SeasonLeader  `json:"top_scorer,omitempty"`
	TopAssists        *SeasonLeader  `json:"top_assists,omitempty"`
	Goalkeeper        *SeasonKeeper  `json:"goalkeeper,omitempty"`
}
type SeasonMetric struct {
	Key           string  `json:"key"`
	Group         string  `json:"group"`
	Value         float64 `json:"value"`
	League        float64 `json:"league"`
	Rank          int     `json:"rank"`
	Clubs         int     `json:"clubs"`
	LowerIsBetter bool    `json:"lower_is_better,omitempty"`
}
type SplitRecord struct {
	Played         int     `json:"played"`
	Wins           int     `json:"wins"`
	Draws          int     `json:"draws"`
	Losses         int     `json:"losses"`
	GoalsFor       int     `json:"goals_for"`
	GoalsAgainst   int     `json:"goals_against"`
	Points         int     `json:"points"`
	PointsPerMatch float64 `json:"points_per_match"`
}
type SeasonLeader struct {
	Player string `json:"player"`
	Value  int    `json:"value"`
}
type SeasonKeeper struct {
	Player      string  `json:"player"`
	Matches     int     `json:"matches"`
	SavePct     float64 `json:"save_pct"`
	CleanSheets int     `json:"clean_sheets"`
}

// Fixture is a scheduled match with no result yet. Time is the kick-off as published
// by the source (local time), empty when unknown.
type Fixture struct {
	Competition string `json:"competition"`
	Round       string `json:"round,omitempty"`
	Date        string `json:"date"`
	Time        string `json:"time,omitempty"`
	HomeTeam    Team   `json:"home_team"`
	AwayTeam    Team   `json:"away_team"`
}

type DataMetadata struct {
	Competition       string   `json:"competition"`
	Season            string   `json:"season"`
	SourceURL         string   `json:"source_url"`
	FetchedAt         string   `json:"fetched_at"`
	LatestMatchDate   string   `json:"latest_match_date"`
	UnavailableFields []string `json:"unavailable_fields"`
	StatisticsSource  string   `json:"statistics_source,omitempty"`
	StatisticsNotice  string   `json:"statistics_notice,omitempty"`
	HistoryNotice     string   `json:"history_notice,omitempty"`
}

// Standing is one league-table row computed from finished matches in the dataset.
type Standing struct {
	Position       int    `json:"position"`
	TeamID         string `json:"team_id"`
	TeamName       string `json:"team_name"`
	LogoURL        string `json:"logo_url"`
	Played         int    `json:"played"`
	Wins           int    `json:"wins"`
	Draws          int    `json:"draws"`
	Losses         int    `json:"losses"`
	GoalsFor       int    `json:"goals_for"`
	GoalsAgainst   int    `json:"goals_against"`
	GoalDifference int    `json:"goal_difference"`
	Points         int    `json:"points"`
}

type Table struct {
	Competition string     `json:"competition"`
	Season      string     `json:"season"`
	SourceURL   string     `json:"source_url"`
	FetchedAt   string     `json:"fetched_at"`
	Standings   []Standing `json:"standings"`
}

// Record is a played/won/drawn/lost summary with goals.
type Record struct {
	Played       int `json:"played"`
	Wins         int `json:"wins"`
	Draws        int `json:"draws"`
	Losses       int `json:"losses"`
	GoalsFor     int `json:"goals_for"`
	GoalsAgainst int `json:"goals_against"`
}

type HistoricMatch struct {
	Season    int    `json:"season"`
	Date      string `json:"date"`
	HomeTeam  string `json:"home_team"`
	AwayTeam  string `json:"away_team"`
	HomeScore int    `json:"home_score"`
	AwayScore int    `json:"away_score"`
	Stadium   string `json:"stadium,omitempty"`
	Goals     []Goal `json:"goals,omitempty"`
}

type HeadToHead struct {
	OpponentID   string `json:"opponent_id"`
	OpponentName string `json:"opponent_name"`
	Record
	LastMatch *HistoricMatch  `json:"last_match"`
	Meetings  []HistoricMatch `json:"meetings,omitempty"` // up to five, newest first
}

// ClubHistory covers Série A only, within FirstSeason–LastSeason of the dataset.
type ClubHistory struct {
	Source        string       `json:"source"`
	SourceURL     string       `json:"source_url"`
	FetchedAt     string       `json:"fetched_at"`
	FirstSeason   int          `json:"first_season"`
	LastSeason    int          `json:"last_season"`
	DatasetName   string       `json:"dataset_name"`
	SeasonsPlayed int          `json:"seasons_played"`
	Titles        []int        `json:"titles"`
	AllTime       Record       `json:"all_time"`
	HeadToHead    []HeadToHead `json:"head_to_head"`
	// Season-by-season detail, the club's Série A scorers and bookings, when available.
	Seasons     []HistorySeason    `json:"seasons"`
	TopScorers  []HistoryScorer    `json:"top_scorers"`
	Discipline  *HistoryDiscipline `json:"discipline,omitempty"`
	Unavailable []string           `json:"unavailable,omitempty"`
}

// LineupPlayer joins the official team sheet with the player's match numbers.
type LineupPlayer struct {
	Number   int     `json:"number"`
	Name     string  `json:"name"`
	Position string  `json:"position"`
	Minutes  int     `json:"minutes"`
	Rating   float64 `json:"rating,omitempty"`
	Goals    int     `json:"goals,omitempty"`
	Assists  int     `json:"assists,omitempty"`
	Yellow   int     `json:"yellow,omitempty"`
	Red      int     `json:"red,omitempty"`
}

type Lineup struct {
	Team        string         `json:"team"`
	Formation   string         `json:"formation,omitempty"`
	Coach       string         `json:"coach,omitempty"`
	Starters    []LineupPlayer `json:"starters"`
	Substitutes []LineupPlayer `json:"substitutes"`
}

type MatchLineups struct {
	Source string `json:"source"`
	Home   Lineup `json:"home"`
	Away   Lineup `json:"away"`
}

// SeasonAverages are per-match means for one club in one season.
type SeasonAverages struct {
	Matches       int     `json:"matches"`
	Possession    float64 `json:"possession"`
	Shots         float64 `json:"shots"`
	ShotsOnTarget float64 `json:"shots_on_target"`
	Corners       float64 `json:"corners"`
	Fouls         float64 `json:"fouls"`
}

type HistorySeason struct {
	Season    int             `json:"season"`
	Position  int             `json:"position,omitempty"`
	Points    int             `json:"points"`
	Record                    // played, wins, draws, losses, goals
	Coaches   []HistoryCoach  `json:"coaches,omitempty"`
	Formation string          `json:"formation,omitempty"`
	Stadium   string          `json:"stadium,omitempty"`
	Averages  *SeasonAverages `json:"averages,omitempty"`
}

type HistoryCoach struct {
	Name    string `json:"name"`
	Matches int    `json:"matches"`
}

type HistoryScorer struct {
	Player    string `json:"player"`
	Goals     int    `json:"goals"`
	Penalties int    `json:"penalties,omitempty"`
	Seasons   string `json:"seasons"`
}

type HistoryBooking struct {
	Player string `json:"player"`
	Yellow int    `json:"yellow"`
	Red    int    `json:"red"`
}

type HistoryDiscipline struct {
	FirstSeason int              `json:"first_season"`
	LastSeason  int              `json:"last_season"`
	Yellow      int              `json:"yellow"`
	Red         int              `json:"red"`
	MostBooked  []HistoryBooking `json:"most_booked"`
}
