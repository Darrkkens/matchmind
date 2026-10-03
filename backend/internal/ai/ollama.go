package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("A IA do MatchMind está indisponível no momento. Verifique se o Ollama está rodando.")
var ErrInvalidResponse = errors.New("O MatchMind não conseguiu validar a resposta da IA. Tente novamente.")

type Client interface {
	Chat(context.Context, string, string) (Answer, error)
	Healthy(context.Context) bool
}
type Ollama struct {
	baseURL, model string
	http           *http.Client
}

func NewOllama(rawURL, model string, timeout time.Duration) (*Ollama, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.TrimSpace(model) == "" || timeout <= 0 {
		return nil, errors.New("invalid Ollama configuration")
	}
	return &Ollama{strings.TrimRight(rawURL, "/"), model, &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (o *Ollama) Chat(ctx context.Context, contextJSON, question string) (Answer, error) {
	// Use one user turn: Gemma templates need not support consecutive user turns.
	// JSON keeps the question and untrusted retrieved data separate from system rules.
	envelope, err := json.Marshal(struct {
		Context  json.RawMessage `json:"CONTEXT"`
		Question string          `json:"QUESTION"`
	}{json.RawMessage(contextJSON), question})
	if err != nil {
		return Answer{}, ErrInvalidResponse
	}
	payload := map[string]any{
		"model": o.model, "stream": false, "options": map[string]any{"temperature": 0.1, "num_predict": 600, "num_ctx": 8192},
		"format":   map[string]any{"type": "object", "additionalProperties": false, "required": []string{"facts", "interpretation", "sources_used"}, "properties": map[string]any{"facts": map[string]any{"type": "string", "minLength": 1}, "interpretation": map[string]any{"type": "string"}, "sources_used": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"team", "recent_matches", "recent_form", "standings", "history", "squad", "trophies"}}}}},
		"messages": []map[string]string{{"role": "system", "content": SystemPrompt}, {"role": "user", "content": string(envelope)}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Answer{}, ErrInvalidResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Answer{}, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := o.http.Do(req)
	if err != nil {
		return Answer{}, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Answer{}, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil || len(data) > 65536 {
		return Answer{}, ErrInvalidResponse
	}
	var response struct {
		Done    bool `json:"done"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(data, &response) != nil || !response.Done {
		return Answer{}, ErrInvalidResponse
	}
	var generated struct {
		Facts          string   `json:"facts"`
		Interpretation *string  `json:"interpretation"`
		SourcesUsed    []string `json:"sources_used"`
	}
	if json.Unmarshal([]byte(response.Message.Content), &generated) != nil || strings.TrimSpace(generated.Facts) == "" || generated.Interpretation == nil || len(generated.Facts)+len(*generated.Interpretation) > 12000 || generated.SourcesUsed == nil {
		return Answer{}, ErrInvalidResponse
	}
	answer := Answer{Answer: "FATO:\n" + strings.TrimSpace(generated.Facts), SourcesUsed: generated.SourcesUsed}
	if interpretation := strings.TrimSpace(*generated.Interpretation); interpretation != "" {
		answer.Answer += "\n\nINTERPRETAÇÃO:\n" + interpretation
	}
	allowed := map[string]bool{"team": true, "recent_matches": true, "recent_form": true, "squad": true, "trophies": true, "standings": true, "history": true}
	seen := map[string]bool{}
	sources := []string{}
	for _, source := range answer.SourcesUsed {
		if !allowed[source] {
			return Answer{}, ErrInvalidResponse
		}
		if !seen[source] {
			sources = append(sources, source)
			seen[source] = true
		}
	}
	answer.SourcesUsed = sources
	return answer, nil
}

// Readiness includes whether the configured model is installed, not merely a live daemon.
func (o *Ollama) Healthy(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	res, err := o.http.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&payload) != nil {
		return false
	}
	for _, m := range payload.Models {
		if m.Name == o.model || m.Name == o.model+":latest" {
			return true
		}
	}
	return false
}
