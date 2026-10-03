package football

import (
	"sort"
	"strings"
)

// CalculateStandings builds a league table from finished matches. Ties are broken
// as in the Brasileirão: points, wins, goal difference, goals scored; the name is
// only a deterministic fallback (head-to-head and disciplinary criteria are not modeled).
func CalculateStandings(teams []Team, matches []Match) []Standing {
	rows := map[string]*Standing{}
	for _, t := range teams {
		rows[t.ID] = &Standing{TeamID: t.ID, TeamName: t.Name, LogoURL: t.LogoURL}
	}
	for _, m := range matches {
		home, away := rows[m.HomeTeam.ID], rows[m.AwayTeam.ID]
		if m.Status != "finished" || home == nil || away == nil || m.HomeScore < 0 || m.AwayScore < 0 {
			continue
		}
		record(home, m.HomeScore, m.AwayScore)
		record(away, m.AwayScore, m.HomeScore)
	}
	table := make([]Standing, 0, len(rows))
	for _, row := range rows {
		row.GoalDifference = row.GoalsFor - row.GoalsAgainst
		table = append(table, *row)
	}
	sort.Slice(table, func(i, j int) bool {
		a, b := table[i], table[j]
		switch {
		case a.Points != b.Points:
			return a.Points > b.Points
		case a.Wins != b.Wins:
			return a.Wins > b.Wins
		case a.GoalDifference != b.GoalDifference:
			return a.GoalDifference > b.GoalDifference
		case a.GoalsFor != b.GoalsFor:
			return a.GoalsFor > b.GoalsFor
		}
		return strings.Compare(normalizeTeam(a.TeamName), normalizeTeam(b.TeamName)) < 0
	})
	for i := range table {
		table[i].Position = i + 1
	}
	return table
}

func record(row *Standing, scored, conceded int) {
	row.Played++
	row.GoalsFor += scored
	row.GoalsAgainst += conceded
	switch {
	case scored > conceded:
		row.Wins++
		row.Points += 3
	case scored == conceded:
		row.Draws++
		row.Points++
	default:
		row.Losses++
	}
}
