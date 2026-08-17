// Command server runs the Marvel Champions web service: REST API,
// WebSocket game streaming and static frontend hosting.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/api"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/rooms"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/store"

	// register game content
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/angel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/captainamerica"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/civilwar"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/core"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/daredevil"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/doctorstrange"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/echo"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/extras"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/galaxysmostwanted"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/goblinfooblin"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/msmarvel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nightcrawler"
)

func main() {
	dbPath := envOr("MC_DB_PATH", "marvelchampions.db")
	listen := envOr("MC_LISTEN", ":3000")
	staticDir := envOr("MC_STATIC_DIR", "web/dist")
	cacheDir := envOr("MC_CACHE_DIR", "cache")

	secret := os.Getenv("MC_JWT_SECRET")
	if secret == "" {
		var buf [32]byte
		if _, err := rand.Read(buf[:]); err != nil {
			log.Fatalf("generate secret: %v", err)
		}
		secret = hex.EncodeToString(buf[:])
		log.Println("MC_JWT_SECRET not set; generated an ephemeral secret (tokens invalidate on restart)")
	}

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	images, err := api.NewImageCache(filepath.Join(cacheDir, "images"))
	if err != nil {
		log.Fatalf("image cache: %v", err)
	}
	server := &api.Server{
		Store:  st,
		Rooms:  rooms.NewManager(st),
		Secret: []byte(secret),
		Images: images,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", server.Router())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Card images are fetched from marvelcdb on demand and cached on
	// disk; content-addressed URLs (/img/cards/{code}.{hash}.png) are
	// immutable, the manifest is always revalidated.
	mux.Handle("GET /img/cards/manifest.json", server.ManifestHandler())
	mux.Handle("/img/cards/", server.ImageHandler())
	// Static frontend with SPA fallback.
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", spaHandler(staticDir, fs))

	log.Printf("listening on %s (db=%s static=%s)", listen, dbPath, staticDir)
	if err := http.ListenAndServe(listen, mux); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// spaHandler serves static files and falls back to index.html for
// client-side routes.
func spaHandler(dir string, fs http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Clean(r.URL.Path)
		full := filepath.Join(dir, path)
		if st, err := os.Stat(full); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

