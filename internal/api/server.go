// Package api exposes the REST + WebSocket surface.
package api

import (
	"context"
	"encoding/json"
	"errors"
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
	// ZhImages serves Chinese card faces seeded into cache/images/zh;
	// codes without a seeded face fall back to Images.
	ZhImages *imageCache
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/authenticate", s.handleAuthenticate)
	mux.HandleFunc("GET /api/v1/whoami", s.auth(s.handleWhoami))

	mux.HandleFunc("GET /api/v1/marvel/cards", s.handleCards)
	mux.HandleFunc("GET /api/v1/marvel/heroes", s.handleHeroes)
	mux.HandleFunc("GET /api/v1/marvel/scenarios", s.handleScenarios)
	mux.HandleFunc("GET /api/v1/locales/manifest", s.handleLocalesManifest)
	mux.HandleFunc("GET /api/v1/locales/{lang}/{hash}", s.handleLocalesVersioned)
	mux.HandleFunc("GET /api/v1/locales/{lang}", s.handleLocales)

	mux.HandleFunc("GET /api/v1/marvel/decks", s.auth(s.handleListDecks))
	mux.HandleFunc("GET /api/v1/marvel/decks/{id}", s.auth(s.handleGetDeck))
	mux.HandleFunc("POST /api/v1/marvel/decks", s.auth(s.handleImportDeck))
	mux.HandleFunc("POST /api/v1/marvel/decks/validate", s.auth(s.handleValidateDeck))
	mux.HandleFunc("DELETE /api/v1/marvel/decks/{id}", s.auth(s.handleDeleteDeck))

	mux.HandleFunc("GET /api/v1/marvel/games", s.auth(s.handleListGames))
	mux.HandleFunc("POST /api/v1/marvel/games", s.auth(s.handleCreateGame))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}", s.auth(s.handleGameView))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/join", s.auth(s.handleJoinGame))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/lobby", s.auth(s.handleLobby))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/kick", s.auth(s.handleKick))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/start", s.auth(s.handleStart))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/answer", s.auth(s.handleAnswer))
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/undo", s.auth(s.handleUndo))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/replay", s.auth(s.handleReplay))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/pile", s.auth(s.handlePileList))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/chat", s.handleChatHistory)
	mux.HandleFunc("POST /api/v1/marvel/games/{id}/chat", s.auth(s.handleChatSend))
	mux.HandleFunc("GET /api/v1/marvel/games/{id}/stream", s.handleStream)

	mux.HandleFunc("GET /api/v1/marvel/campaigns", s.auth(s.handleListCampaigns))
	mux.HandleFunc("POST /api/v1/marvel/campaigns", s.auth(s.handleCreateCampaign))
	mux.HandleFunc("GET /api/v1/marvel/campaigns/{id}", s.auth(s.handleGetCampaign))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/join", s.auth(s.handleJoinCampaign))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/kick", s.auth(s.handleKickCampaign))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/start", s.auth(s.handleStartCampaign))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/play", s.auth(s.handlePlayCampaign))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/choice", s.auth(s.handleCampaignChoice))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/market", s.auth(s.handleCampaignMarket))
	mux.HandleFunc("POST /api/v1/marvel/campaigns/{id}/heal", s.auth(s.handleCampaignHeal))

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
		claims, err := s.parseToken(tokenStr)
		if err != nil {
			// Distinguish expiry from bad signatures so clients can react
			// (and logs make sense) without guessing.
			if errors.Is(err, jwt.ErrTokenExpired) {
				writeErr(w, http.StatusUnauthorized, "token expired")
			} else {
				writeErr(w, http.StatusUnauthorized, "invalid token")
			}
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

// pathGame resolves the {id} path value as an opaque game token. On miss it
// writes the error response and returns ok=false; sequential ids never match.
func (s *Server) pathGame(w http.ResponseWriter, r *http.Request) (int64, bool) {
	gameID, err := s.Store.GameIDByToken(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "game not found")
		return 0, false
	}
	return gameID, true
}

func (s *Server) issueToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprint(userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.Secret)
}

// parseToken validates a signed token and returns its claims.
func (s *Server) parseToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.Secret, nil
	})
	if err != nil || !token.Valid {
		if err == nil {
			err = jwt.ErrTokenInvalidClaims
		}
		return nil, err
	}
	return claims, nil
}

// parseSubject validates a token and returns its subject (user id).
func (s *Server) parseSubject(tokenStr string) (string, bool) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return "", false
	}
	sub, err := claims.GetSubject()
	return sub, err == nil
}

var _ = engine.DB // referenced via handlers
