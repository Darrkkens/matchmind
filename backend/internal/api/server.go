package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"matchmind/internal/ai"
	"matchmind/internal/football"
	"matchmind/internal/service"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Server struct {
	teams     *service.TeamService
	chat      *service.ChatService
	origins   map[string]bool
	mu        sync.Mutex
	window    time.Time
	requests  int
	inference chan struct{}
}

func New(teams *service.TeamService, chat *service.ChatService, origins []string) http.Handler {
	s := &Server{teams: teams, chat: chat, origins: map[string]bool{}, inference: make(chan struct{}, 1)}
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			s.origins[o] = true
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/team/resolve", s.resolve)
	mux.HandleFunc("POST /api/chat", s.ask)
	mux.HandleFunc("GET /api/standings", s.standings)
	mux.HandleFunc("GET /api/lineups/{ref}", s.lineups)
	return s.middleware(mux)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fail(w, 415, "Use Content-Type: application/json.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err = d.Decode(v)
	if err == nil {
		var extra any
		if d.Decode(&extra) != io.EOF {
			err = errors.New("multiple JSON values")
		}
	}
	if err != nil {
		var size *http.MaxBytesError
		if errors.As(err, &size) {
			fail(w, 413, "O corpo da requisição é grande demais.")
		} else {
			fail(w, 400, "Requisição JSON inválida ou com campos inesperados.")
		}
		return false
	}
	return true
}
func serviceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, football.ErrNotFound), errors.Is(err, football.ErrLineupsUnavailable):
		fail(w, 404, err.Error())
	case errors.Is(err, football.ErrInvalidInput), errors.Is(err, football.ErrAmbiguous), errors.Is(err, service.ErrValidation):
		fail(w, 400, err.Error())
	case errors.Is(err, ai.ErrUnavailable):
		fail(w, 503, ai.ErrUnavailable.Error())
	case errors.Is(err, ai.ErrInvalidResponse):
		fail(w, 502, ai.ErrInvalidResponse.Error())
	case errors.Is(err, football.ErrProviderUnavailable):
		w.Header().Set("Retry-After", "15")
		fail(w, 503, football.ErrProviderUnavailable.Error())
	case errors.Is(err, football.ErrDatasetUnavailable), errors.Is(err, football.ErrProviderData), errors.Is(err, football.ErrStandingsUnavailable):
		fail(w, 502, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		fail(w, 504, "A requisição demorou demais. Tente novamente.")
	case errors.Is(err, context.Canceled):
		fail(w, 408, "Requisição cancelada.")
	default:
		slog.Error("request failed", "error", err)
		fail(w, 500, "Não foi possível concluir a requisição.")
	}
}
func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input string `json:"input"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	result, err := s.teams.Resolve(r.Context(), req.Input)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (s *Server) ask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID   string `json:"team_id"`
		Question string `json:"question"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	select {
	case s.inference <- struct{}{}:
		defer func() { <-s.inference }()
	default:
		w.Header().Set("Retry-After", "5")
		fail(w, 429, "O MatchMind está respondendo outra pergunta. Tente novamente em instantes.")
		return
	}
	answer, err := s.chat.Ask(r.Context(), req.TeamID, req.Question)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, answer)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	providerReady := make(chan bool, 1)
	go func() { providerReady <- s.teams.Healthy(ctx) }()
	ollama := s.chat.AI.Healthy(ctx)
	provider := <-providerReady
	status := "ok"
	if !provider || !ollama {
		status = "degraded"
	}
	writeJSON(w, 200, map[string]any{"status": status, "ollama": ollama, "football_provider": provider, "data_source": s.teams.Source})
}

var lineupRef = regexp.MustCompile(`^[0-9]{1,10}$`)

// lineups accepts only the opaque numeric reference returned in a snapshot.
func (s *Server) lineups(w http.ResponseWriter, r *http.Request) {
	ref := r.PathValue("ref")
	if !lineupRef.MatchString(ref) {
		fail(w, 400, "Referência de escalação inválida.")
		return
	}
	result, err := s.teams.Lineups(r.Context(), ref)
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) standings(w http.ResponseWriter, r *http.Request) {
	table, err := s.teams.Standings(r.Context())
	if err != nil {
		serviceError(w, err)
		return
	}
	writeJSON(w, 200, table)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !s.origins[origin] {
				fail(w, 403, "Origem não permitida.")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(204)
			return
		}
		// A fixed global budget avoids unbounded per-IP memory and spoofed forwarding headers.
		s.mu.Lock()
		if time.Since(s.window) >= time.Minute {
			s.window = time.Now()
			s.requests = 0
		}
		s.requests++
		limited := s.requests > 120
		s.mu.Unlock()
		if limited {
			w.Header().Set("Retry-After", "60")
			fail(w, 429, "Muitas requisições. Aguarde um minuto.")
			return
		}
		defer func() {
			if recover() != nil {
				slog.Error("recovered HTTP handler panic")
				fail(w, 500, "Não foi possível concluir a requisição.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
