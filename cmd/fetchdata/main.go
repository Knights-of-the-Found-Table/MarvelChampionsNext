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
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}

	body, err := getJSON(client, baseURL+"/packs/", *pause)
	if err != nil {
		log.Fatalf("fetch pack list: %v", err)
	}
	var packs []packInfo
	if err := json.Unmarshal(body, &packs); err != nil {
		log.Fatalf("decode pack list: %v", err)
	}

	if err := writeRaw(filepath.Join(*outDir, "packs.json"), body); err != nil {
		log.Fatalf("write packs.json: %v", err)
	}
	log.Printf("fetched pack list: %d packs", len(packs))

	// Keep the snapshot deterministic regardless of API ordering.
	for _, p := range packs {
		cardsBody, err := getJSON(client, baseURL+"/cards/"+p.Code, *pause)
		if err != nil {
			log.Fatalf("fetch cards for %s: %v", p.Code, err)
		}
		var pretty json.RawMessage
		if err := json.Unmarshal(cardsBody, &pretty); err != nil {
			log.Fatalf("decode cards for %s: %v", p.Code, err)
		}
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err != nil {
			log.Fatalf("re-encode cards for %s: %v", p.Code, err)
		}
		if err := writeRaw(filepath.Join(*outDir, p.Code+".json"), indented); err != nil {
			log.Fatalf("write %s.json: %v", p.Code, err)
		}
		log.Printf("fetched %-6s %-30s (%d known cards)", p.Code, p.Name, p.Known)
	}

	log.Printf("snapshot complete: %d packs written to %s", len(packs), *outDir)
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
