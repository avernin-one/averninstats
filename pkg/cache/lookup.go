package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
)

const (
	TypeBlock     = "block"
	TypeItem      = "item"
	TypeEntity    = "entity"
	TypeCustom    = "custom"
	TypePlayer    = "player"
	TypeHighscore = "highscore"
)

type Lookup struct {
	Block  []string `json:"block"`
	Item   []string `json:"item"`
	Entity []string `json:"entity"`
	Custom []string `json:"stats"`
}

var filePath = filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "lookup.json")

// Load reads the lookup from the cache file.
// Returns an error if the file does not exist or cannot be parsed.
func Load() (*Lookup, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read lookup file: %w", err)
	}

	var l Lookup
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("decode lookup file: %w", err)
	}

	return &l, nil
}

// Save writes the lookup to the cache file.
func (l *Lookup) Save() error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o775); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lookup: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o664); err != nil {
		return fmt.Errorf("write lookup file: %w", err)
	}

	log.Info().Str("path", filePath).Msg("lookup saved")
	return nil
}

// AnyEmpty reports whether any category list is empty.
func (l *Lookup) AnyEmpty() bool {
	return len(l.Block) == 0 || len(l.Item) == 0 || len(l.Entity) == 0 || len(l.Custom) == 0
}
