package service

import (
	"context"
	"errors"
	"matchmind/internal/ai"
	"matchmind/internal/football"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrValidation = errors.New("team_id must be valid and question must contain 1–1000 characters")
var validID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,100}$`)

type ChatService struct {
	Teams *TeamService
	AI    ai.Client
}

func (s *ChatService) Ask(ctx context.Context, id, question string) (ai.Answer, error) {
	question = strings.TrimSpace(question)
	if !validID.MatchString(id) || question == "" || !utf8.ValidString(question) || utf8.RuneCountInString(question) > 1000 || strings.ContainsFunc(question, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) {
		return ai.Answer{}, ErrValidation
	}
	snapshot, err := s.Teams.Snapshot(ctx, id)
	if err != nil {
		return ai.Answer{}, err
	}
	// Simulation questions get the app's computed numbers; the model only explains them.
	// Prefer the AI-reviewed simulation the user just ran; otherwise a quick one without the
	// AI step, since a second model call would not fit the chat timeout on CPU.
	var sim *football.Simulation
	if snapshot.NextMatch != nil && mentionsAny(question, ai.SimulationWords) {
		if sim = s.Teams.CachedSimulation(id, snapshot.NextMatch); sim == nil {
			if sim, err = s.Teams.Simulate(ctx, id, simulationRuns(question), false); err != nil {
				sim = nil
			}
		}
	}
	contextJSON, err := ai.BuildContextWithSimulation(snapshot, question, sim)
	if err != nil {
		return ai.Answer{}, err
	}
	answer, err := s.AI.Chat(ctx, contextJSON, question)
	if err != nil {
		return answer, err
	}
	// Exact simulation numbers come from the app, not from the model's retelling.
	if sim != nil {
		answer.Answer = ai.SimulationSummary(sim) + "\n\n" + answer.Answer
		if !slices.Contains(answer.SourcesUsed, "simulation") {
			answer.SourcesUsed = append(answer.SourcesUsed, "simulation")
		}
	}
	// The model may ignore the sample-data rule, so the warning is added here.
	if m := snapshot.DataMetadata; m != nil && strings.HasSuffix(m.StatisticsSource, "-teste") && slices.Contains(answer.SourcesUsed, "recent_matches") {
		answer.Answer += "\n\nATENÇÃO: as estatísticas das partidas (posse, finalizações, escanteios etc.) vêm da chave de teste da API-Futebol e são dados de exemplo, não os números reais."
	}
	return answer, nil
}

var runsPattern = regexp.MustCompile(`(\d[\d.]*)\s*(?:simula|vezes|rodadas|execu)`)

// simulationRuns reads "10.000 simulações" / "50 vezes" from the question; default 50.
func simulationRuns(question string) int {
	if m := runsPattern.FindStringSubmatch(strings.ToLower(question)); m != nil {
		if n, err := strconv.Atoi(strings.ReplaceAll(m[1], ".", "")); err == nil && n >= 1 && n <= 10000 {
			return n
		}
	}
	return DefaultSimulationRuns
}

func mentionsAny(text string, words []string) bool {
	text = strings.ToLower(text)
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}
