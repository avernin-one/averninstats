package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/frontend"
	"github.com/avernin-one/averninstats/pkg/stats"
	"github.com/avernin-one/averninstats/pkg/translations"
	"github.com/rs/zerolog/log"
)

func main() {
	start := time.Now()

	// Init config.
	cfg := config.Init()

	// Copy embedded static files to output directory.
	frontend.Copy()

	// Download language files and build/load the category lookup map.
	lookup, err := translations.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to process languages")
	}

	// Load player cache.
	playerCache := cache.NewPlayerCache()

	// Set up processor and clear stale output directories.
	proc := stats.New(lookup)

	// Process every per-player stats JSON file.
	entries, err := os.ReadDir(cfg.StatsSourceDir)
	if err != nil {
		log.Fatal().Err(err).Str("dir", cfg.StatsSourceDir).Msg("cannot read stats source directory")
	}

	current := 0
	skipped := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			skipped++
			log.Debug().Bool("is_dir", entry.IsDir()).Str("name", entry.Name()).Msg("skipping non-JSON file")
			continue
		}

		current++
		uuid := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(cfg.StatsSourceDir, entry.Name())

		log.Info().
			Str("file", entry.Name()).
			Int("current", current).
			Int("skipped", skipped).
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
	if err := playerCache.Save(); err != nil {
		log.Warn().Err(err).Msg("failed to save player cache")
	}

	log.Info().TimeDiff("duration", time.Now(), start).Msg("finished")
}
