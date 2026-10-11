package cache

import (
	"fmt"

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

	// stat -> category, built lazily from the lists above.
	types map[string]string
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

// Returns the category of stat. Blocks take precedence over items and items
// over entities, as some names exist in multiple categories.
func (l *Lookup) GetType(stat string) (string, error) {
	if l.types == nil {
		l.buildIndex()
	}

	if t, ok := l.types[stat]; ok {
		return t, nil
	}

	return "", fmt.Errorf(`stat "%s" not found`, stat)
}

func (l *Lookup) buildIndex() {
	l.types = make(map[string]string, len(l.Block)+len(l.Item)+len(l.Entity))

	// Insert in reverse precedence so higher priority categories win.
	for _, e := range []struct {
		typ   string
		names []string
	}{
		{TypeEntity, l.Entity},
		{TypeItem, l.Item},
		{TypeBlock, l.Block},
	} {
		for _, name := range e.names {
			l.types[name] = e.typ
		}
	}
}
