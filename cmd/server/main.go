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
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/api"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/dotenv"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/mirror"
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
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/hulk"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/msmarvel"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/mutantgenesis"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nextevolution"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/nightcrawler"
	_ "github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine/cards/thor"
)

func main() {
	// Optional repo-root .env (KEY=VALUE) with e.g. the image-mirror
	// credentials; real environment variables take precedence.
	if n, err := dotenv.Load(".env"); err != nil {
		log.Printf(".env: %v", err)
	} else if n > 0 {
		log.Printf("loaded %d variables from .env", n)
	}

	dbPath := envOr("MC_DB_PATH", "marvelchampions.db")
	listen := envOr("MC_LISTEN", ":3000")
	staticDir := envOr("MC_STATIC_DIR", "web/dist")
	cacheDir := envOr("MC_CACHE_DIR", "cache")

	// Optional Simplified Chinese translation overlay: a directory of pack
	// JSON files (tools/zh/out). Untranslated cards keep English values.
	if zhDir := os.Getenv("MC_ZH_DIR"); zhDir != "" {
		n, err := engine.ApplyChinese(engine.DB, zhDir)
		if err != nil {
			log.Fatalf("zh translations: %v", err)
		}
		engine.RelabelScenarios(engine.DB)
		log.Printf("zh translations: %d cards overlaid from %s", n, zhDir)
	}

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

	// Image sources, one chain per language. Mirrors follow the face
	// convention ({base}a = A face, {base}b = B face, requested as
	// /bundles/cards/{code}.png), so both a configured IMAGE_MIRROR and the
	// Chinese root (ZH_IMAGE_MIRROR) resolve by convention; only bare
	// marvelcdb.com — the default when IMAGE_MIRROR is unset — needs its
	// legacy per-face paths.
	imgSources := mirror.SourcesFromEnv()
	for _, src := range imgSources.Default {
		log.Printf("image source (default): %s", src.Name())
	}
	for _, src := range imgSources.Zh {
		log.Printf("image source (zh): %s", src.Name())
	}

	images, err := api.NewImageCacheWithPaths(
		filepath.Join(cacheDir, "images"),
		api.DefaultImagePathFor(imgSources.DefaultIsMirror),
		imgSources.Default...,
	)
	if err != nil {
		log.Fatalf("image cache: %v", err)
	}
	// The zh cache serves locally seeded faces plus ZH_IMAGE_MIRROR
	// fetches by convention path; codes it cannot resolve fall back to the
	// default chain.
	zhImages, err := api.NewImageCacheWithPaths(filepath.Join(cacheDir, "images", "zh"), api.ConventionImagePath, imgSources.Zh...)
	if err != nil {
		log.Fatalf("zh image cache: %v", err)
	}

	// Prewarm mirror-backed caches in the background (MC_PREWARM_IMAGES=1
	// forces both even against bare marvelcdb, =0 disables): filling the
	// manifests makes every image URL content-addressed. The caches
	// persist in MC_CACHE_DIR, so repeat boots only fetch what is missing.
	if prewarm := envOr("MC_PREWARM_IMAGES", "auto"); prewarm != "0" {
		prewarmDefault := prewarm == "1" || imgSources.DefaultIsMirror
		prewarmZh := prewarm == "1" || len(imgSources.Zh) > 0
		if prewarmDefault || prewarmZh {
			// Every card code, faces included: the manifest then covers
			// all URLs the frontend can build. Codes a source lacks are
			// counted as missing (zh falls back to the default chain).
			codes := make([]string, 0, len(engine.DB.All()))
			for _, def := range engine.DB.All() {
				codes = append(codes, def.Code)
			}
			if prewarmDefault {
				log.Printf("images: prewarming %d card images (default) in the background", len(codes))
				go api.PrewarmImages(images, codes, 4, 25*time.Millisecond)
			}
			if prewarmZh {
				log.Printf("images: prewarming %d card images (zh) in the background", len(codes))
				go api.PrewarmImages(zhImages, codes, 4, 25*time.Millisecond)
			}
		}
	}
	server := &api.Server{
		Store:    st,
		Rooms:    rooms.NewManager(st),
		Secret:   []byte(secret),
		Images:   images,
		ZhImages: zhImages,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", server.Router())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Card images are fetched on demand through the shared source chain
	// and cached on disk; content-addressed URLs (/img/cards/{code}.{hash}.png)
	// are immutable, the manifest is always revalidated. The zh routes
	// serve the zh cache (locally seeded faces first, then the same chain);
	// anything unresolved falls back to the English cache.
	mux.Handle("GET /img/cards/manifest.json", server.ManifestHandler())
	mux.Handle("/img/cards/", server.ImageHandler())
	mux.Handle("GET /img/cards/zh/manifest.json", server.ZhManifestHandler())
	mux.Handle("/img/cards/zh/", server.ZhImageHandler())
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
