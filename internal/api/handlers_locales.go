package api

import (
	"net/http"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// handleLocales serves the engine message catalog for one language:
// GET /api/v1/locales/{lang} -> { "<message key>": "<format string>", ... }.
// This is the single source of truth for client-side rendering of logs,
// prompts, choice labels and game-over reasons; the server itself stays
// language-neutral (see internal/engine/i18n.go).
func (s *Server) handleLocales(w http.ResponseWriter, r *http.Request) {
	lang := engine.Lang(r.PathValue("lang"))
	if lang != engine.LangEn && lang != engine.LangZh {
		writeErr(w, http.StatusNotFound, "unknown locale")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, engine.Messages(lang))
}
