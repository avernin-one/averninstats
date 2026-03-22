package stats

// manifest.go — writes _manifest.json files for each output category.
// The frontend has no directory listing, so it needs a manifest to know
// which stat/player JSON files exist.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/rs/zerolog/log"
)

// highscoreManifest lists all stat names present in the highscore directory.
type highscoreManifest struct {
	Stats []string `json:"stats"`
}

// statManifest lists all stat names for blocks/items/entities.
type statManifest struct {
	Stats []string `json:"stats"`
}

// playerManifest lists all player names.
type playerManifest struct {
	Players []string `json:"players"`
}

// WriteManifests writes _manifest.json files into each category output directory.
// Must be called after Flush so all JSON files are already written.
func (p *Processor) WriteManifests() error {
	if err := p.writeHighscoreManifest(); err != nil {
		return err
	}
	for _, cat := range []string{cache.TypeBlock, cache.TypeItem, cache.TypeEntity} {
		if err := p.writeStatManifest(cat); err != nil {
			return err
		}
	}
	return p.writePlayerManifest()
}

func (p *Processor) writeHighscoreManifest() error {
	names := make([]string, 0, len(p.highscores))
	for name := range p.highscores {
		names = append(names, name)
	}
	sort.Strings(names)
	return writeManifestJSON(
		filepath.Join(p.cfg.OutputDir, cache.TypeHighscore, "_manifest.json"),
		highscoreManifest{Stats: names},
	)
}

func (p *Processor) writeStatManifest(category string) error {
	var data StatScores
	switch category {
	case cache.TypeBlock:
		data = p.scores.Block
	case cache.TypeItem:
		data = p.scores.Item
	case cache.TypeEntity:
		data = p.scores.Entity
	default:
		return fmt.Errorf("unknown category %q", category)
	}

	names := make([]string, 0, len(data))
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)
	return writeManifestJSON(
		filepath.Join(p.cfg.OutputDir, category, "_manifest.json"),
		statManifest{Stats: names},
	)
}

func (p *Processor) writePlayerManifest() error {
	names := make([]string, 0, len(p.players))
	for _, player := range p.players {
		if player.Name != "" {
			names = append(names, player.Name)
		}
	}
	sort.Strings(names)
	return writeManifestJSON(
		filepath.Join(p.cfg.OutputDir, cache.TypePlayer, "_manifest.json"),
		playerManifest{Players: names},
	)
}

func writeManifestJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return fmt.Errorf("create dir for manifest: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o664); err != nil {
		return fmt.Errorf("write manifest %q: %w", path, err)
	}
	log.Debug().Str("path", path).Msg("manifest written")
	return nil
}
