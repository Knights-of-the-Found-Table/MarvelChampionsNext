// Package api exposes the REST + WebSocket surface.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"
)

type Server struct {
	Store  *store.Store
	Rooms  *rooms.Manager
	Secret []byte
	// Images serves card images with on-demand marvelcdb fetching.
	Images *imageCache
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/authenticate", s.handleAuthenticate)
	mux.HandleFunc("GET /api/v1/whoami", s.auth(s.handleWhoami))

	mux.HandleFunc("GET /api/v1/marvel/cards", s.handleCards)
	mux.HandleFunc("GET /api/v1/marvel/heroes", s.handleHeroes)
	mux.HandleFunc("GET /api/v1/marvel/scenarios", s.handleScenarios)

	mux.HandleFunc("GET /api/v1/marvel/decks", s.auth(s.handleListDecks))
	mux.HandleFunc("POST /api/v1/marvel/decks", s.auth(s.handleImportDeck))
	mux.HandleFunc("DELETE /api/v1/marvel/decks/{id}", s.auth(s.handleDeleteDeck))

	mux.HandleFunc("GET /api/v1/marvel/games", s.auth(s.handleListGames))
	mux.HandleFunc("POST /api/v1/marvel/games", s.auth(s.handleCreateGame))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}", s.auth(s.handleGameView))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/join", s.auth(s.handleJoinGame))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/answer", s.auth(s.handleAnswer))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/undo", s.auth(s.handleUndo))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/replay", s.auth(s.handleReplay))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/stream", s.handleStream)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}

// ---------------------------------------------------------------- helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type ctxKey int

const userIDKey ctxKey = 1

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := bearerToken(r)
		if tokenStr == "" {
			writeErr(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims := jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return s.Secret, nil
		})
		if err != nil || !token.Valid {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		uid, err := claims.GetSubject()
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid subject")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, uid)))
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// WebSocket clients cannot set headers in browsers: allow ?token=.
	if r.URL.Query().Get("token") != "" {
		return r.URL.Query().Get("token")
	}
	return ""
}

func userID(r *http.Request) string {
	v, _ := r.Context().Value(userIDKey).(string)
	return v
}

func pathID(r *http.Request) (int64, error) {
	idStr := r.PathValue("id")
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}

func (s *Server) issueToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprint(userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Secret)
}

// parseSubject validates a token and returns its subject (user id).
func (s *Server) parseSubject(tokenStr string) (string, bool) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.Secret, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	sub, err := claims.GetSubject()
	return sub, err == nil
}

var _ = engine.DB // referenced via handlers
