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

	// Init config.
	cfg := config.Init()

	// Download language files and build/load the category lookup map.
	lookup, err := translations.Run(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to process languages")
	}

	// Load player cache.
	playerCache := cache.NewPlayerCache(cfg)

	// Set up processor and clear stale output directories.
	proc := stats.New(cfg, lookup)

	// Process every per-player stats JSON file.
	entries, err := os.ReadDir(cfg.StatsSourceDir)
	if err != nil {
		log.Fatal().Err(err).Str("dir", cfg.StatsSourceDir).Msg("cannot read stats source directory")
	}

	total := countJSONFiles(entries)
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
			log.Warn().Err(err).Str("file", entry.Name()).Msg("skipping unreadable file")
		}
	}

	// Resolve names, write player files, write global stat files.
	if err := proc.Flush(playerCache); err != nil {
		log.Fatal().Err(err).Msg("failed to flush output")
	}

	if err := proc.WriteManifests(); err != nil {
		log.Warn().Err(err).Msg("failed to write manifests")
	}

	// Persist updated player cache.
	if err := playerCache.SaveToFile(); err != nil {
		log.Warn().Err(err).Msg("failed to save player cache")
	}

	log.Info().TimeDiff("duration", start, time.Now()).Msg("finished")
}

func countJSONFiles(entries []os.DirEntry) int {
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			n++
		}
	}
	return n
}
