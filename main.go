package main

import (
	"time"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/frontend"
	"github.com/avernin-one/averninstats/pkg/processor"
	"github.com/avernin-one/averninstats/pkg/translations"
	"github.com/rs/zerolog/log"
)

func main() {
	start := time.Now()

	// Init config.
	config.Init()

	// Copy embedded static files to output directory.
	frontend.New().Copy()

	// Download language files and build/load the category lookup map.
	// This step is critical to create the LookupMaps which are later used to
	// process the player stats.
	lookup, err := translations.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to process languages")
	}

	// Load player cache.
	playerCache := cache.NewPlayerCache()

	// Set up processor and clear stale output directories.
	proc := processor.New(lookup, playerCache)
	proc.Process()

	playerCache.Save()

	log.Info().TimeDiff("duration", time.Now(), start).Msg("finished")
}
