package processor

// manifest.go - writes _manifest.json files for each output category.
// The frontend has no directory listing, so it needs a manifest to know
// which stat/player JSON files exist.

import (
	"fmt"
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

	names := make([]string, 0, len(p.Highscores))
	for name := range p.Highscores {
		names = append(names, name)
	}

	sort.Strings(names)

	if err := utils.SaveJSONFile(outFile, names); err != nil {
		log.Error().Err(err).Str("category", cache.TypeHighscore).Msg("failed to write highscore manifest file")
	}
}

func (p *Processor) writeStatsManifest() {
	for category, data := range p.Scores {
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

	// we can't use strings.Sort() here cause it is case-sensitive and will not
	// sort as expected.
	sort.Slice(p.PlayerNames, func(i, j int) bool {
		return utils.LessLower(p.PlayerNames[i], p.PlayerNames[j])
	})

	if err := utils.SaveJSONFile(outFile, p.PlayerNames); err != nil {
		log.Error().Err(err).Str("category", cache.TypePlayer).Msg("failed to write player manifest file")
	}
}

func (p *Processor) WritePlayer(player *Player) {
	outFile := filepath.Join(config.Get().OutputDir, cache.TypePlayer, fmt.Sprintf("%s.json", player.Name))
	if err := utils.SaveJSONFile(outFile, player); err != nil {
		log.Error().Err(err).Str("player", player.Name).Msg("failed to write player file")
	}
}

func (p *Processor) WriteStats() {
	for scoreType, scoreData := range p.Scores {
		outFolder := filepath.Join(config.Get().OutputDir, scoreType)

		for fileName, data := range scoreData {
			outFile := filepath.Join(outFolder, fmt.Sprintf("%s.json", fileName))
			if err := utils.SaveJSONFile(outFile, data); err != nil {
				log.Error().Err(err).Str("file", outFile).Msg("failed to write stats file")
			}
		}
	}
}

func (p *Processor) WriteHighscore() {
	outFile := filepath.Join(config.Get().OutputDir, cache.TypeHighscore, fmt.Sprintf("%s.json", cache.TypeHighscore))
	if err := utils.SaveJSONFile(outFile, p.Highscores); err != nil {
		log.Error().Err(err).Str("file", outFile).Msg("failed to write highscore file")
	}
}
