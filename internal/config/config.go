// # internal/config/config.go
package config

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	GrammarsPath string   `toml:"grammars_path"`
	WatchPaths   []string `toml:"watch_paths"`
	Exclude      Exclude  `toml:"exclude"`
	Watch        Watch    `toml:"watch"`
	Output       Output   `toml:"output"`
	Alerts       Alerts   `toml:"alerts"`
}

type Exclude struct {
	Dirs    []string `toml:"dirs"`
	Files   []string `toml:"files"`
	Symbols []string `toml:"symbols"` // Prefixes to ignore (e.g., self., ctx.)
}

type Watch struct {
	Debounce time.Duration `toml:"debounce"`
}

type Output struct {
	DOT string `toml:"dot"`
	TSV string `toml:"tsv"`
}

type Alerts struct {
	Beep     bool `toml:"beep"`
	Terminal bool `toml:"terminal"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}

	// Default debounce if not set
	if cfg.Watch.Debounce == 0 {
		cfg.Watch.Debounce = 500 * time.Millisecond
	}

	if len(cfg.WatchPaths) == 0 {
		cfg.WatchPaths = []string{"."}
	}

	return &cfg, nil
}
