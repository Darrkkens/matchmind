package service

import (
	"context"
	"errors"
	"matchmind/internal/ai"
	"regexp"
	"slices"
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
	contextJSON, err := ai.BuildContext(snapshot, question)
	if err != nil {
		return ai.Answer{}, err
	}
	answer, err := s.AI.Chat(ctx, contextJSON, question)
	if err != nil {
		return answer, err
	}
	// The model may ignore the sample-data rule, so the warning is added here.
	if m := snapshot.DataMetadata; m != nil && strings.HasSuffix(m.StatisticsSource, "-teste") && slices.Contains(answer.SourcesUsed, "recent_matches") {
		answer.Answer += "\n\nATENÇÃO: as estatísticas das partidas (posse, finalizações, escanteios etc.) vêm da chave de teste da API-Futebol e são dados de exemplo, não os números reais."
	}
	return answer, nil
}
