// Command fetchdata downloads the Marvel Champions card database snapshot
// from the marvelcdb.com public API into internal/engine/data/packs/.
//
// The snapshot is committed to the repository so that builds and the image
// fetcher work offline and are reproducible.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/logx"
)

const baseURL = "https://marvelcdb.com/api/public"

type packInfo struct {
	Name      string `json:"name"`
	Code      string `json:"code"`
	Position  int    `json:"position"`
	Available string `json:"available"`
	Known     int    `json:"known"`
	Total     int    `json:"total"`
}

func main() {
	outDir := flag.String("out", filepath.Join("internal", "engine", "data", "packs"), "output directory for the data snapshot")
	pause := flag.Duration("pause", 500*time.Millisecond, "pause between API requests to stay polite")
	flag.Parse()
	logx.Init()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		logx.Fatal("create output dir", "error", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}

	body, err := getJSON(client, baseURL+"/packs/", *pause)
	if err != nil {
		logx.Fatal("fetch pack list", "error", err)
	}
	var packs []packInfo
	if err := json.Unmarshal(body, &packs); err != nil {
		logx.Fatal("decode pack list", "error", err)
	}

	if err := writeRaw(filepath.Join(*outDir, "packs.json"), body); err != nil {
		logx.Fatal("write packs.json", "error", err)
	}
	slog.Info("fetched pack list", "packs", len(packs))

	// Keep the snapshot deterministic regardless of API ordering.
	for _, p := range packs {
		cardsBody, err := getJSON(client, baseURL+"/cards/"+p.Code, *pause)
		if err != nil {
			logx.Fatal("fetch cards", "pack", p.Code, "error", err)
		}
		var pretty json.RawMessage
		if err := json.Unmarshal(cardsBody, &pretty); err != nil {
			logx.Fatal("decode cards", "pack", p.Code, "error", err)
		}
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err != nil {
			logx.Fatal("re-encode cards", "pack", p.Code, "error", err)
		}
		if err := writeRaw(filepath.Join(*outDir, p.Code+".json"), indented); err != nil {
			logx.Fatal("write pack file", "pack", p.Code, "error", err)
		}
		slog.Info("fetched pack", "code", p.Code, "name", p.Name, "known", p.Known)
	}

	slog.Info("snapshot complete", "packs", len(packs), "dir", *outDir)
}

func getJSON(client *http.Client, url string, pause time.Duration) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "marvelchampions-go/fetchdata")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	time.Sleep(pause)
	return body, err
}

func writeRaw(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
