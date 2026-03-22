package stats

import (
	"sort"

	"github.com/avernin-one/averninstats/pkg/cache"
)

var categories = []string{
	cache.TypeHighscore,
	cache.TypeBlock,
	cache.TypeItem,
	cache.TypeEntity,
	cache.TypePlayer,
}

// sortedKeys returns the keys of list in descending order (highest first).
func sortedKeys(list ScoreList) []int {
	keys := make([]int, 0, len(list))
	for k := range list {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] > keys[j] })
	return keys
}

// trimScoreList removes the lowest-scoring entries so at most max unique
// score values are kept.
func trimScoreList(list ScoreList, max int) {
	keys := sortedKeys(list)
	if len(keys) <= max {
		return
	}
	for _, k := range keys[max:] {
		delete(list, k)
	}
}
