package football

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"time"
)

// Simulation is a Monte Carlo estimate for the next fixture from the selected club's
// side. It is a statistical model over the available data, never a certainty.
type Simulation struct {
	Runs          int                `json:"runs"`
	Seed          uint64             `json:"seed"`
	Fixture       Fixture            `json:"fixture"`
	ClubID        string             `json:"club_id"`
	Win           float64            `json:"win_pct"`
	Draw          float64            `json:"draw_pct"`
	Loss          float64            `json:"loss_pct"`
	ExpectedClub  float64            `json:"expected_goals_club"`
	ExpectedOpp   float64            `json:"expected_goals_opponent"`
	Scorelines    []Scoreline        `json:"top_scorelines"`
	Factors       []SimulationFactor `json:"factors"`
	ClubRecent    []string           `json:"club_last5"`
	OpponentLast5 []string           `json:"opponent_last5"`
	Notes         []string           `json:"notes"`
}

type Scoreline struct {
	Club     int     `json:"club"`
	Opponent int     `json:"opponent"`
	Percent  float64 `json:"percent"`
}

// SimulationFactor records how one input moved each side's expected goals
// (multipliers: 1.05 = +5%). Unavailable factors stay at 1 with a reason.
type SimulationFactor struct {
	Key       string  `json:"key"`
	Club      float64 `json:"club"`
	Opponent  float64 `json:"opponent"`
	Detail    string  `json:"detail"`
	Available bool    `json:"available"`
}

// SimulationInput is everything the model reads; built from the season dataset and history.
type SimulationInput struct {
	Fixture   Fixture
	ClubID    string
	Matches   []Match // all finished league matches this season, newest first
	Standings []Standing
	History   *HeadToHead // Série A record against the opponent, when known
	// Optional season detail (shots, home/away splits) for both sides, from a local source.
	ClubSeason, OpponentSeason *SeasonStats
	// Optional adjustment from the AI analyst, applied before the random draws.
	Analyst *SimulationAdjustment
}

// SimulationAdjustment is the AI analyst's judgement: bounded multipliers on each side's
// expected goals and the reason, written for the user.
type SimulationAdjustment struct {
	Club     float64 `json:"club"`
	Opponent float64 `json:"opponent"`
	Reason   string  `json:"reason"`
	Model    string  `json:"model"`
}

// MaxAnalystSwing bounds the AI analyst's adjustment (±15%).
const MaxAnalystSwing = 0.15

const (
	maxSimulationRuns = 10000
	formWeight        = 0.3  // share of the last-5 scoring/conceding rate blended into the season rate
	h2hWeight         = 0.15 // largest swing from a fully one-sided head-to-head record
	h2hPrior          = 15.0 // meetings needed for the head-to-head to carry half its weight
	dayNoise          = 0.2  // log-normal spread of each side's goal rate per simulated match
	shrinkMatches     = 5.0  // pseudo-matches at the league average, so early seasons are not extreme
)

// Simulate runs the model. The seed is derived from the fixture and run count, so the
// same request always gives the same numbers (the UI and the AI explain one result).
func Simulate(in SimulationInput, runs int) (*Simulation, error) {
	if runs < 1 || runs > maxSimulationRuns {
		return nil, ErrSimulationRuns
	}
	f := in.Fixture
	clubHome := f.HomeTeam.ID == in.ClubID
	club, opp := f.HomeTeam, f.AwayTeam
	if !clubHome {
		club, opp = f.AwayTeam, f.HomeTeam
	}
	if club.ID != in.ClubID {
		return nil, ErrNotFound
	}
	rows := map[string]Standing{}
	totalGoals, totalPlayed := 0, 0
	for _, s := range in.Standings {
		rows[s.TeamID] = s
		totalGoals += s.GoalsFor
		totalPlayed += s.Played
	}
	if totalPlayed == 0 {
		return nil, ErrInsufficientData
	}
	mu := float64(totalGoals) / float64(totalPlayed) // goals per team per match
	homeGoals, awayGoals, n := 0, 0, 0
	for _, m := range in.Matches {
		homeGoals += m.HomeScore
		awayGoals += m.AwayScore
		n++
	}
	homeEdge, awayEdge := 1.0, 1.0
	if n > 0 && homeGoals+awayGoals > 0 {
		homeEdge = float64(homeGoals) / float64(n) / mu
		awayEdge = float64(awayGoals) / float64(n) / mu
	}
	rate := func(goals, played int) float64 {
		return (float64(goals) + mu*shrinkMatches) / (float64(played) + shrinkMatches)
	}
	cs, os := rows[club.ID], rows[opp.ID]
	clubAtt, clubDef := rate(cs.GoalsFor, cs.Played)/mu, rate(cs.GoalsAgainst, cs.Played)/mu
	oppAtt, oppDef := rate(os.GoalsFor, os.Played)/mu, rate(os.GoalsAgainst, os.Played)/mu
	clubEdge, oppEdge := awayEdge, homeEdge
	if clubHome {
		clubEdge, oppEdge = homeEdge, awayEdge
	}
	lambdaClub := mu * clubAtt * oppDef * clubEdge
	lambdaOpp := mu * oppAtt * clubDef * oppEdge

	sim := &Simulation{Runs: runs, Fixture: f, ClubID: club.ID, Factors: []SimulationFactor{}, Notes: []string{}}
	seasonMetric := func(st *SeasonStats, key string) (SeasonMetric, bool) {
		if st == nil {
			return SeasonMetric{}, false
		}
		for _, m := range st.Metrics {
			if m.Key == key && m.League > 0 {
				return m, true
			}
		}
		return SeasonMetric{}, false
	}
	pct := func(m float64) string { return fmt.Sprintf("%+.0f%%", (m-1)*100) }
	sim.Factors = append(sim.Factors, SimulationFactor{Key: "season", Club: 1, Opponent: 1, Available: true,
		Detail: fmt.Sprintf("Base: média da liga %.2f gols por time e jogo, ataque e defesa de cada clube na temporada; mando de campo %s para o mandante", mu, pct(homeEdge))})

	// Season detail: scoring and chance creation/concession from the season source, blended
	// half-and-half with the score-only rates above (shots on target are steadier than goals).
	index := func(st *SeasonStats, goalsKey, shotsKey string) (float64, bool) {
		g, ok1 := seasonMetric(st, goalsKey)
		sh, ok2 := seasonMetric(st, shotsKey)
		if !ok1 || !ok2 {
			return 0, false
		}
		return 0.5*g.Value/g.League + 0.5*sh.Value/sh.League, true
	}
	ca, ok1 := index(in.ClubSeason, "goals", "shots_on_target")
	cdf, ok2 := index(in.ClubSeason, "goals_against", "shots_on_target_against")
	oa, ok3 := index(in.OpponentSeason, "goals", "shots_on_target")
	odf, ok4 := index(in.OpponentSeason, "goals_against", "shots_on_target_against")
	if ok1 && ok2 && ok3 && ok4 {
		mc := clamp(math.Sqrt(ca*odf/(clubAtt*oppDef)), 0.85, 1.15)
		mo := clamp(math.Sqrt(oa*cdf/(oppAtt*clubDef)), 0.85, 1.15)
		lambdaClub, lambdaOpp = lambdaClub*mc, lambdaOpp*mo
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "season_detail", Club: round2(mc), Opponent: round2(mo), Available: true,
			Detail: fmt.Sprintf("Temporada (%s, até %s): ataque %.2f e defesa %.2f de %s × ataque %.2f e defesa %.2f de %s (1,00 = média da liga, gols e chutes no alvo)", in.ClubSeason.Source, in.ClubSeason.AsOf, ca, cdf, club.Name, oa, odf, opp.Name)})
	} else {
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "season_detail", Club: 1, Opponent: 1, Detail: "Estatísticas detalhadas da temporada indisponíveis (configure FBREF_DIR)"})
	}
	// Venue: each side's own home or away scoring relative to its overall rate, beyond the league-wide edge.
	split := func(st *SeasonStats, home bool, edge float64) (float64, *SplitRecord, bool) {
		if st == nil || st.Matches == 0 {
			return 1, nil, false
		}
		r := st.Away
		if home {
			r = st.Home
		}
		g, ok := seasonMetric(st, "goals")
		if r == nil || r.Played == 0 || !ok || g.Value <= 0 || edge <= 0 {
			return 1, nil, false
		}
		return clamp(1+0.5*((float64(r.GoalsFor)/float64(r.Played))/g.Value/edge-1), 0.85, 1.15), r, true
	}
	vc, rc, okc := split(in.ClubSeason, clubHome, clubEdge)
	vo, ro, oko := split(in.OpponentSeason, !clubHome, oppEdge)
	if okc && oko {
		lambdaClub, lambdaOpp = lambdaClub*vc, lambdaOpp*vo
		where := map[bool]string{true: "em casa", false: "fora"}
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "venue", Club: round2(vc), Opponent: round2(vo), Available: true,
			Detail: fmt.Sprintf("Mando: %s %s %dV %dE %dD (%d:%d), %s %s %dV %dE %dD (%d:%d)", club.Name, where[clubHome], rc.Wins, rc.Draws, rc.Losses, rc.GoalsFor, rc.GoalsAgainst, opp.Name, where[!clubHome], ro.Wins, ro.Draws, ro.Losses, ro.GoalsFor, ro.GoalsAgainst)})
	} else {
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "venue", Club: 1, Opponent: 1, Detail: "Campanha em casa e fora indisponível"})
	}

	// 1. Last five matches of each side, blended into the season rates.
	recent := func(id string) (gf, ga, played int, seq []string) {
		for _, m := range in.Matches {
			if m.HomeTeam.ID != id && m.AwayTeam.ID != id {
				continue
			}
			for_, against := m.HomeScore, m.AwayScore
			if m.AwayTeam.ID == id {
				for_, against = m.AwayScore, m.HomeScore
			}
			gf, ga, played = gf+for_, ga+against, played+1
			switch {
			case for_ > against:
				seq = append(seq, "W")
			case for_ < against:
				seq = append(seq, "L")
			default:
				seq = append(seq, "D")
			}
			if played == 5 {
				break
			}
		}
		return
	}
	cgf, cga, cn, cseq := recent(club.ID)
	ogf, oga, on, oseq := recent(opp.ID)
	sim.ClubRecent, sim.OpponentLast5 = nonNil(cseq), nonNil(oseq)
	formMult := func(recentRate, seasonRate float64) float64 {
		if seasonRate <= 0 {
			return 1
		}
		return clamp(1+formWeight*(recentRate/seasonRate-1), 0.8, 1.2)
	}
	if cn > 0 && on > 0 {
		mc := formMult(float64(cgf)/float64(cn), clubAtt*mu) * formMult(float64(oga)/float64(on), oppDef*mu)
		mo := formMult(float64(ogf)/float64(on), oppAtt*mu) * formMult(float64(cga)/float64(cn), clubDef*mu)
		mc, mo = clamp(mc, 0.8, 1.2), clamp(mo, 0.8, 1.2)
		lambdaClub, lambdaOpp = lambdaClub*mc, lambdaOpp*mo
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "form", Club: round2(mc), Opponent: round2(mo), Available: true,
			Detail: fmt.Sprintf("Últimos %d jogos: %s %d:%d · últimos %d: %s %d:%d", cn, club.Name, cgf, cga, on, opp.Name, ogf, oga)})
	} else {
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "form", Club: 1, Opponent: 1, Detail: "Sem jogos recentes suficientes"})
	}

	// 2. Rest: days since each side's last league match, and congestion in the prior 10 days.
	rest := func(id string) (days int, crowded int, ok bool) {
		kickoff, err := time.Parse("2006-01-02", f.Date)
		if err != nil {
			return 0, 0, false
		}
		for _, m := range in.Matches {
			if m.HomeTeam.ID != id && m.AwayTeam.ID != id {
				continue
			}
			d, err := time.Parse("2006-01-02", m.Date)
			if err != nil || !d.Before(kickoff) {
				continue
			}
			gap := int(kickoff.Sub(d).Hours() / 24)
			if !ok {
				days, ok = gap, true
			}
			if gap <= 10 {
				crowded++
			}
		}
		return
	}
	fatigue := func(days, crowded int) float64 {
		m := 1.0
		switch {
		case days <= 2:
			m = 0.92
		case days == 3:
			m = 0.96
		}
		if crowded >= 3 {
			m *= 0.97
		}
		return m
	}
	cd, cc, cok := rest(club.ID)
	od, oc, ook := rest(opp.ID)
	if cok && ook {
		fc, fo := fatigue(cd, cc), fatigue(od, oc)
		// A tired side scores less and concedes a little more.
		mc, mo := fc*(1+(1-fo)/2), fo*(1+(1-fc)/2)
		lambdaClub, lambdaOpp = lambdaClub*mc, lambdaOpp*mo
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "rest", Club: round2(mc), Opponent: round2(mo), Available: true,
			Detail: fmt.Sprintf("Descanso: %s %d dias (%d jogos em 10 dias), %s %d dias (%d jogos em 10 dias); 3 dias ou menos pesam", club.Name, cd, cc, opp.Name, od, oc)})
		sim.Notes = append(sim.Notes, "O descanso considera só jogos desta liga; copas e outras competições não estão nos dados.")
	} else {
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "rest", Club: 1, Opponent: 1, Detail: "Datas dos jogos anteriores indisponíveis"})
	}

	// 3. Injuries: no source in this app yet.
	sim.Factors = append(sim.Factors, SimulationFactor{Key: "injuries", Club: 1, Opponent: 1, Detail: "Lesões e suspensões não estão disponíveis em nenhuma fonte configurada; o fator não altera a simulação"})

	// 4 and 7. Head-to-head: Série A history plus this season's meetings.
	w, d, l, gf, ga := 0, 0, 0, 0, 0
	if h := in.History; h != nil {
		w, d, l, gf, ga = h.Wins, h.Draws, h.Losses, h.GoalsFor, h.GoalsAgainst
	}
	for _, m := range in.Matches {
		home, away := m.HomeTeam.ID, m.AwayTeam.ID
		if !(home == club.ID && away == opp.ID) && !(home == opp.ID && away == club.ID) {
			continue
		}
		for_, against := m.HomeScore, m.AwayScore
		if away == club.ID {
			for_, against = m.AwayScore, m.HomeScore
		}
		gf, ga = gf+for_, ga+against
		switch {
		case for_ > against:
			w++
		case for_ < against:
			l++
		default:
			d++
		}
	}
	if meetings := w + d + l; meetings > 0 {
		share := float64(3*w+d) / float64(3*meetings)
		weight := float64(meetings) / (float64(meetings) + h2hPrior)
		m := 1 + h2hWeight*weight*(2*share-1)
		lambdaClub, lambdaOpp = lambdaClub*m, lambdaOpp/m
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "head_to_head", Club: round2(m), Opponent: round2(1 / m), Available: true,
			Detail: fmt.Sprintf("Confrontos (Série A desde 2003 e esta temporada): %d jogos, %dV %dE %dD, gols %d:%d", meetings, w, d, l, gf, ga)})
	} else {
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "head_to_head", Club: 1, Opponent: 1, Detail: "Sem confrontos registrados"})
	}

	// AI analyst: a bounded judgement over the same facts, applied like any other factor.
	if a := in.Analyst; a != nil {
		mc := clamp(a.Club, 1-MaxAnalystSwing, 1+MaxAnalystSwing)
		mo := clamp(a.Opponent, 1-MaxAnalystSwing, 1+MaxAnalystSwing)
		lambdaClub, lambdaOpp = lambdaClub*mc, lambdaOpp*mo
		sim.Factors = append(sim.Factors, SimulationFactor{Key: "ai_analyst", Club: round2(mc), Opponent: round2(mo), Available: true, Detail: a.Reason})
	}

	// 5. Randomness: each simulated match gets its own day-to-day swing before the Poisson draw.
	sim.Factors = append(sim.Factors, SimulationFactor{Key: "randomness", Club: 1, Opponent: 1, Available: true,
		Detail: fmt.Sprintf("Cada simulação varia a força de cada lado (±%.0f%% típico) e sorteia os gols por Poisson", dayNoise*100)})

	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%d", f.Date, f.HomeTeam.ID, f.AwayTeam.ID, club.ID, runs)))
	sim.Seed = binary.LittleEndian.Uint64(sum[:8])
	rng := rand.New(rand.NewPCG(sim.Seed, sim.Seed^0x9e3779b97f4a7c15))
	counts := map[[2]int]int{}
	wins, draws := 0, 0
	for range runs {
		a := poisson(rng, lambdaClub*math.Exp(rng.NormFloat64()*dayNoise-dayNoise*dayNoise/2))
		b := poisson(rng, lambdaOpp*math.Exp(rng.NormFloat64()*dayNoise-dayNoise*dayNoise/2))
		counts[[2]int{a, b}]++
		switch {
		case a > b:
			wins++
		case a == b:
			draws++
		}
	}
	share := func(k int) float64 { return math.Round(float64(k)/float64(runs)*1000) / 10 }
	sim.Win, sim.Draw = share(wins), share(draws)
	sim.Loss = math.Round((100-sim.Win-sim.Draw)*10) / 10
	sim.ExpectedClub, sim.ExpectedOpp = round2(lambdaClub), round2(lambdaOpp)
	for k, c := range counts {
		sim.Scorelines = append(sim.Scorelines, Scoreline{Club: k[0], Opponent: k[1], Percent: share(c)})
	}
	sort.Slice(sim.Scorelines, func(i, j int) bool {
		a, b := sim.Scorelines[i], sim.Scorelines[j]
		if a.Percent != b.Percent {
			return a.Percent > b.Percent
		}
		return a.Club*10+a.Opponent < b.Club*10+b.Opponent
	})
	sim.Scorelines = sim.Scorelines[:min(5, len(sim.Scorelines))]
	if runs < 1000 {
		sim.Notes = append(sim.Notes, fmt.Sprintf("Com %d simulações as porcentagens variam vários pontos; 10.000 dão um resultado mais estável.", runs))
	}
	return sim, nil
}

// poisson draws by inversion; fine for football goal rates (λ well under 10).
func poisson(rng *rand.Rand, lambda float64) int {
	limit, p, k := math.Exp(-lambda), 1.0, 0
	for {
		p *= rng.Float64()
		if p <= limit || k > 20 {
			return k
		}
		k++
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func round2(v float64) float64        { return math.Round(v*100) / 100 }
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
