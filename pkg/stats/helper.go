package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/rs/zerolog/log"
)

// IndexEntry is written as _index.json in each category directory.
type IndexEntry struct {
	Title string `json:"title"`
}

var categories = []string{
	cache.TypeHighscore,
	cache.TypeBlock,
	cache.TypeItem,
	cache.TypeEntity,
	cache.TypePlayer,
}

// WriteIndexFiles creates one _index.json per category and optionally clears
// the category directory first (controlled by cfg.NoDelete).
func (p *Processor) WriteIndexFiles() error {
	for _, category := range categories {
		dir := filepath.Join(p.cfg.OutputDir, category)

		if !p.cfg.NoDelete {
			log.Info().Str("dir", dir).Msg("removing category directory")
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("remove %q: %w", dir, err)
			}
		}

		if err := saveJSON(filepath.Join(dir, "_index.json"), IndexEntry{Title: category}); err != nil {
			return fmt.Errorf("write index for %q: %w", category, err)
		}
	}
	return nil
}

// replaceUUIDWithName replaces all occurrences of uuid with name in every
// score list across highscores and block/item/entity scores. This is necessary
// because scores are aggregated during ProcessFile when the player name is not
// yet known — the UUID is used as a placeholder and swapped out here.
func (p *Processor) replaceUUIDWithName(uuid, name string) {
	replaceInScoreList := func(list ScoreList) {
		for score, players := range list {
			for i, v := range players {
				if v == uuid {
					list[score][i] = name
				}
			}
		}
	}

	for _, scoreList := range p.highscores {
		replaceInScoreList(scoreList)
	}

	for _, statScores := range []StatScores{p.scores.Block, p.scores.Item, p.scores.Entity} {
		for _, actionScores := range statScores {
			for _, scoreList := range actionScores {
				replaceInScoreList(scoreList)
			}
		}
	}
}

// (highest score first).
func sortedKeys(list ScoreList) []int {
	keys := make([]int, 0, len(list))
	for k := range list {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	return keys
}
