package stats

// manifest.go - writes _manifest.json files for each output category.
// The frontend has no directory listing, so it needs a manifest to know
// which stat/player JSON files exist.

import (
	"path/filepath"
	"sort"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/utils"
	"github.com/rs/zerolog/log"
)

const manifestFileName = "_manifest.json"

// WriteManifests writes _manifest.json files into each category output directory.
// Must be called after Flush so all JSON files are already written.
func (p *Processor) WriteManifests() {
	p.writeHighscoreManifest()
	p.writePlayerManifest()
	p.writeStatsManifest()
}

func (p *Processor) writeHighscoreManifest() {
	outFile := filepath.Join(config.Get().OutputDir, cache.TypeHighscore, manifestFileName)

	names := make([]string, 0, len(p.highscores))
	for name := range p.highscores {
		names = append(names, name)
	}

	sort.Strings(names)

	if err := utils.SaveJSONFile(outFile, names); err != nil {
		log.Error().Err(err).Str("category", cache.TypeHighscore).Msg("failed to write highscore manifest file")
	}
}

func (p *Processor) writeStatsManifest() {
	cats := map[string]StatScores{
		cache.TypeBlock:  p.scores.Block,
		cache.TypeItem:   p.scores.Item,
		cache.TypeEntity: p.scores.Entity,
	}

	for category, data := range cats {
		outFile := filepath.Join(config.Get().OutputDir, category, manifestFileName)

		names := make([]string, 0, len(data))
		for name := range data {
			names = append(names, name)
		}

		sort.Strings(names)

		if err := utils.SaveJSONFile(outFile, names); err != nil {
			log.Error().Err(err).Str("category", category).Msg("failed to write manifest file")
		}
	}
}

func (p *Processor) writePlayerManifest() {
	outFile := filepath.Join(config.Get().OutputDir, cache.TypePlayer, manifestFileName)

	names := make([]string, 0, len(p.players))
	for _, player := range p.players {
		if player.Name != "" {
			names = append(names, player.Name)
		}
	}

	sort.Strings(names)

	if err := utils.SaveJSONFile(outFile, names); err != nil {
		log.Error().Err(err).Str("category", cache.TypePlayer).Msg("failed to write player manifest file")
	}
}
