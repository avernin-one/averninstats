package cache

import (
	"fmt"
	"slices"

	"github.com/avernin-one/averninstats/pkg/utils"
)

const (
	TypeBlock     = "block"
	TypeItem      = "item"
	TypeEntity    = "entity"
	TypeCustom    = "custom"
	TypePlayer    = "player"
	TypeHighscore = "highscore"
)

// Holds the category membership lists used to classify raw Minecraft
// stat keys. Built once from the en-gb language file and cached on disk.
type Lookup struct {
	Block  []string `json:"block"`
	Item   []string `json:"item"`
	Entity []string `json:"entity"`
	Custom []string `json:"stats"`
}

// Reads the lookup from cachePath.
// Returns an error if the file is missing or malformed.
func LoadLookup() (*Lookup, error) {
	var l Lookup
	if err := utils.ReadJSONFile(LookupFile(), &l); err != nil {
		return nil, fmt.Errorf("read lookup cache: %w", err)
	}

	return &l, nil
}

func (l *Lookup) Save() error {
	return utils.SaveJSONFile(LookupFile(), l)
}

// Reports whether any category list has no entries.
func (l *Lookup) AnyEmpty() bool {
	return len(l.Block) == 0 || len(l.Item) == 0 || len(l.Entity) == 0 || len(l.Custom) == 0
}

func (l *Lookup) GetType(stat string) (string, error) {
	if slices.Contains(l.Block, stat) {
		return TypeBlock, nil
	}

	if slices.Contains(l.Item, stat) {
		return TypeItem, nil
	}

	if slices.Contains(l.Entity, stat) {
		return TypeEntity, nil
	}

	return "", fmt.Errorf(`stat "%s" not found`, stat)
}
