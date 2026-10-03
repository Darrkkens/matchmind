package football

import "math"

func CalculateForm(teamID string, matches []Match) RecentForm {
	f := RecentForm{Sequence: []string{}}
	for _, m := range matches {
		if m.Status != "finished" || (m.HomeTeam.ID != teamID && m.AwayTeam.ID != teamID) || m.HomeScore < 0 || m.AwayScore < 0 {
			continue
		}
		scored, conceded := m.HomeScore, m.AwayScore
		if m.AwayTeam.ID == teamID {
			scored, conceded = conceded, scored
		}
		f.Played++
		f.GoalsScored += scored
		f.GoalsConceded += conceded
		switch {
		case scored > conceded:
			f.Wins++
			f.Sequence = append(f.Sequence, "W")
		case scored == conceded:
			f.Draws++
			f.Sequence = append(f.Sequence, "D")
		default:
			f.Losses++
			f.Sequence = append(f.Sequence, "L")
		}
	}
	if f.Played > 0 {
		f.AverageGoals = math.Round(float64(f.GoalsScored)/float64(f.Played)*100) / 100
		f.PointsPercentage = math.Round(float64(f.Wins*3+f.Draws)/float64(f.Played*3)*1000) / 10
	}
	return f
}
