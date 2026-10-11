package processor

import (
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

// Removes the "minecraft:" prefix from any given key and return
// the stripped string back.
func trimNamespace(key string) string {
	if s, ok := strings.CutPrefix(key, "minecraft:"); ok {
		return s
	}

	log.Warn().Str("key", key).Msg("stats key missing namespace prefix")

	return key
}

// Removes the lowest-scoring entries so at most max unique
// score values are kept.
func trimScoreList(list map[int][]string, keep int) {
	keys := sortedKeys(list)
	if len(keys) <= keep {
		return
	}

	for _, k := range keys[keep:] {
		delete(list, k)
	}
}

// Returns the keys of list in descending order (highest first).
func sortedKeys(list map[int][]string) []int {
	keys := make([]int, 0, len(list))
	for k := range list {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	return keys
}

// Counts the gold/silver/bronze placements of every player across all
// highscores.
func (p *Processor) computeMedals() map[string]Medals {
	medals := make(map[string]Medals)

	for _, scoreList := range p.Highscores {
		for rank, key := range sortedKeys(scoreList) {
			if rank >= 3 {
				break // only first 3 (gold/silver/bronze)
			}

			for _, name := range scoreList[key] {
				m := medals[name]
				switch rank {
				case 0:
					m.Gold++
				case 1:
					m.Silver++
				case 2:
					m.Bronze++
				}
				medals[name] = m
			}
		}
	}

	return medals
}
