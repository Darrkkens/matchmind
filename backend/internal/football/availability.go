package football

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// MatchAvailability holds likely absences of both sides for the next fixture.
type MatchAvailability struct {
	Club     *Availability `json:"club,omitempty"`
	Opponent *Availability `json:"opponent,omitempty"`
}

// NextMatchAvailability applies the Brasileirão card rules to a club's last league match:
// a red card suspends for the next match; every third yellow (3rd, 6th, 9th…) too. Season
// totals come from the season source; when they predate the last match, its yellow is added.
// Tribunal (STJD) decisions and injuries are not known.
func NextMatchAvailability(teamID string, last *Match, season *SeasonStats) *Availability {
	if season == nil {
		return nil
	}
	out := &Availability{Suspended: []Absence{}, AtRisk: []string{}}
	byName := func(name string) (PlayerBookings, bool) {
		best, bestDiff, found := PlayerBookings{}, math.MaxInt, false
		for _, b := range season.Bookings {
			if diff, ok := playerDistance(cleanName(name), cleanName(b.Player)); ok && diff < bestDiff {
				best, bestDiff, found = b, diff, true
			}
		}
		return best, found
	}
	suspended := map[string]bool{}
	switch {
	case last == nil:
		out.Note = "Sem último jogo na liga; suspensões não verificadas."
	case last.Cards == nil:
		out.Note = "Cartões do último jogo indisponíveis; suspensões não verificadas."
	default:
		side := "home"
		opponent := last.AwayTeam.Name
		if last.AwayTeam.ID == teamID {
			side, opponent = "away", last.HomeTeam.Name
		}
		totalsIncludeLast := season.AsOf >= last.Date
		for _, c := range last.Cards {
			if c.Side != side {
				continue
			}
			b, known := byName(c.Player)
			name := c.Player
			if known {
				name = b.Player
			}
			if suspended[name] {
				continue
			}
			absence := Absence{Player: name, MinutesPct: b.MinutesPct, OnOff: b.OnOff}
			switch {
			case c.Color == "red":
				absence.Reason = fmt.Sprintf("expulso no último jogo (contra %s, %s)", opponent, last.Date)
			case known:
				yellows := b.Yellow
				if !totalsIncludeLast {
					yellows++
				}
				if yellows == 0 || yellows%3 != 0 {
					continue
				}
				absence.Reason = fmt.Sprintf("%dº cartão amarelo no último jogo (contra %s)", yellows, opponent)
			default:
				continue
			}
			suspended[name] = true
			out.Suspended = append(out.Suspended, absence)
		}
		if !totalsIncludeLast {
			out.Note = fmt.Sprintf("Totais de cartões até %s; o amarelo do último jogo (%s) foi somado.", season.AsOf, last.Date)
		}
	}
	// One yellow away, most used players first (they matter most if they miss a match).
	risk := []PlayerBookings{}
	for _, b := range season.Bookings {
		if b.Yellow%3 == 2 && !suspended[b.Player] {
			risk = append(risk, b)
		}
	}
	sort.SliceStable(risk, func(i, j int) bool { return risk[i].MinutesPct > risk[j].MinutesPct })
	for _, b := range risk {
		out.AtRisk = append(out.AtRisk, b.Player)
	}
	return out
}

// absenceWeight is how much one suspended player lowers his team's strength: more for
// regulars and for players the team does better with (on/off), at most ~7%.
func absenceWeight(a Absence) float64 {
	return 0.04*clamp(a.MinutesPct/100, 0, 1) + 0.03*clamp(a.OnOff, 0, 1)
}

// suspensionFactor turns both sides' absences into multipliers: a weakened side scores
// less and concedes a little more. Each side's total is capped at 12%.
func suspensionFactor(club, opp *Availability, clubName, oppName string) (SimulationFactor, bool) {
	if club == nil || opp == nil {
		return SimulationFactor{Key: "suspensions", Club: 1, Opponent: 1, Detail: "Suspensões por cartão não verificadas (configure FBREF_DIR e ALMANACSTATS)"}, false
	}
	weight := func(a *Availability) (float64, string) {
		w, names := 0.0, []string{}
		for _, s := range a.Suspended {
			w += absenceWeight(s)
			names = append(names, s.Player)
		}
		if len(names) == 0 {
			return 0, "nenhum"
		}
		return math.Min(w, 0.12), strings.Join(names, ", ")
	}
	wc, nc := weight(club)
	wo, no := weight(opp)
	mc := (1 - 0.6*wc) * (1 + 0.4*wo)
	mo := (1 - 0.6*wo) * (1 + 0.4*wc)
	return SimulationFactor{Key: "suspensions", Club: round2(mc), Opponent: round2(mo), Available: true,
		Detail: fmt.Sprintf("Suspensos por cartão: %s: %s; %s: %s", clubName, nc, oppName, no) + noEffect(mc, mo, "nenhum suspenso relevante")}, true
}
