// Package dotenv loads a .env file (KEY=VALUE lines) into the process
// environment, so a repo-root .env can hold the image-mirror credentials
// for local dev as well as docker compose (which reads the same file via
// env_file). Variables already present in the real environment win; a
// missing file is not an error.
package dotenv

import (
	"fmt"
	"os"
	"strings"
)

// Load sets variables from path that are not already in the environment
// and returns how many were set.
func Load(path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	set := 0
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k == "" || os.Getenv(k) != "" {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return set, fmt.Errorf("%s: %w", k, err)
		}
		set++
	}
	return set, nil
}
