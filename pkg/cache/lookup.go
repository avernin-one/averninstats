package cache

import (
	"encoding/json"
	"fmt"
	"os"

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

// Lookup holds the category membership lists used to classify raw Minecraft
// stat keys. Built once from the en-gb language file and cached on disk.
type Lookup struct {
	Block  []string `json:"block"`
	Item   []string `json:"item"`
	Entity []string `json:"entity"`
	Custom []string `json:"stats"`

	// blockSet, itemSet, entitySet provide O(1) membership checks.
	// They are populated lazily on first call to Contains().
	blockSet  map[string]struct{}
	itemSet   map[string]struct{}
	entitySet map[string]struct{}
}

// LoadLookup reads the lookup from cachePath.
// Returns an error if the file is missing or malformed.
func LoadLookup(cachePath string) (*Lookup, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("read lookup cache %q: %w", cachePath, err)
	}

	var l Lookup
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("decode lookup cache %q: %w", cachePath, err)
	}

	l.buildSets()
	return &l, nil
}

// Save persists the Lookup to cachePath.
func (l *Lookup) Save(cachePath string) error {
	if err := os.MkdirAll(dirOf(cachePath), 0o775); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lookup: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0o664); err != nil {
		return fmt.Errorf("write lookup cache: %w", err)
	}

	log.Info().Str("path", cachePath).Msg("lookup saved")
	return nil
}

// AnyEmpty reports whether any category list has no entries.
func (l *Lookup) AnyEmpty() bool {
	return len(l.Block) == 0 || len(l.Item) == 0 ||
		len(l.Entity) == 0 || len(l.Custom) == 0
}

// ContainsBlock reports whether stat is a known block key. O(1).
func (l *Lookup) ContainsBlock(stat string) bool {
	_, ok := l.blockSet[stat]
	return ok
}

// ContainsItem reports whether stat is a known item key. O(1).
func (l *Lookup) ContainsItem(stat string) bool {
	_, ok := l.itemSet[stat]
	return ok
}

// ContainsEntity reports whether stat is a known entity key. O(1).
func (l *Lookup) ContainsEntity(stat string) bool {
	_, ok := l.entitySet[stat]
	return ok
}

// buildSets converts the slice-based lists to maps for O(1) lookups.
func (l *Lookup) buildSets() {
	l.blockSet = toSet(l.Block)
	l.itemSet = toSet(l.Item)
	l.entitySet = toSet(l.Entity)
}

func toSet(s []string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}
