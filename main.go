package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/stats"
	"github.com/avernin-one/averninstats/pkg/translations"
	"github.com/rs/zerolog/log"
)

func main() {
	start := time.Now()

	cfg := config.Get()

	// 1. Download all language files and build / load the lookup map.
	lookup, err := translations.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to process languages")
	}

	// 2. Load the player cache (resolved names + skin metadata).
	playerCache := cache.NewPlayerCache()

	// 3. Set up the stats processor, clear old output directories.
	proc := stats.New(lookup)

	// 4. Read and process every per-player stats JSON file.
	entries, err := os.ReadDir(cfg.StatsSourceDir)
	if err != nil {
		log.Fatal().Err(err).Str("dir", cfg.StatsSourceDir).Msg("cannot read stats source directory")
	}

	total := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			total++
		}
	}

	current := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		current++

		uuid := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(cfg.StatsSourceDir, entry.Name())

		log.Info().
			Str("file", entry.Name()).
			Int("current", current).
			Int("total", total).
			Msg("processing stats file")

		if err := proc.ProcessFile(path, uuid); err != nil {
			log.Warn().Err(err).Str("file", entry.Name()).Msg("skipping unreadable stats file")
		}
	}

	// 5. Resolve names, assign medals, apply playtime filter,
	//    write per-player JSON files and render skin images.
	if err := proc.WritePlayerFiles(playerCache); err != nil {
		log.Fatal().Err(err).Msg("failed to write player files")
	}

	// 6. Write global highscore and block/item/entity stat files.
	if _, err := proc.Flush(); err != nil {
		log.Fatal().Err(err).Msg("failed to flush stat files")
	}

	// 7. Persist the updated player cache.
	if err := playerCache.SaveToFile(); err != nil {
		log.Warn().Err(err).Msg("failed to save player cache")
	}

	log.Info().TimeDiff("duration", start, time.Now()).Msg("finished")
}
