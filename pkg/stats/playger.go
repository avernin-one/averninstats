package stats

import (
	"fmt"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/rs/zerolog/log"
)

// WritePlayerFiles resolves each player's name via the cache (fetching from
// the Mojang API if necessary), assigns medals, filters by minimum playtime,
// writes the JSON file and saves skin images.
func (p *Processor) WritePlayerFiles(pc *cache.PlayerCache) error {
	for _, player := range p.players {
		cp, err := pc.GetOrFetch(player.UUID)
		if err != nil {
			log.Warn().Err(err).Str("uuid", player.UUID).Msg("failed to resolve player, skipping")
			continue
		}
		player.Name = cp.Name

		// Score maps were populated with UUIDs as placeholders (the name was
		// not known yet during ProcessFile). Replace them with the real name now.
		p.replaceUUIDWithName(player.UUID, player.Name)

		p.setMedals(player)

		if !p.meetsPlaytimeRequirement(player) {
			log.Warn().Str("name", player.Name).Msg("player below minimum playtime, skipping")
			continue
		}

		path := filepath.Join(p.cfg.OutputDir, cache.TypePlayer, fmt.Sprintf("%s.json", player.Name))

		if err := saveJSON(path, player); err != nil {
			log.Warn().Err(err).Str("player", player.Name).Msg("failed to write player file")
		}

		pc.EnsureSkin(cp, p.cfg.OutputDir)
	}
	return nil
}

// setMedals counts how many global highscore top-1/2/3 positions this player holds.
func (p *Processor) setMedals(player *Player) {
	player.Medals = Medals{}

	for _, scoreList := range p.highscores {
		keys := sortedKeys(scoreList)
		for rank, key := range keys {
			for _, name := range scoreList[key] {
				if name != player.Name {
					continue
				}
				switch rank {
				case 0:
					player.Medals.Gold++
				case 1:
					player.Medals.Silver++
				case 2:
					player.Medals.Bronze++
				}
			}
		}
	}
}

// meetsPlaytimeRequirement returns true if the player has enough playtime,
// or if they already appear in any highscore / stat ranking.
func (p *Processor) meetsPlaytimeRequirement(player *Player) bool {
	if p.playerHasHighscore(player) || p.playerIsRanked(player) {
		return true
	}

	ticks, ok := player.Stats["play_time"]
	if !ok {
		return false
	}

	return ticks/20/60 >= p.cfg.MinPlayTime
}

func (p *Processor) playerHasHighscore(player *Player) bool {
	for _, scoreList := range p.highscores {
		for _, names := range scoreList {
			for _, name := range names {
				if name == player.Name {
					return true
				}
			}
		}
	}
	return false
}

func (p *Processor) playerIsRanked(player *Player) bool {
	for _, statScores := range []StatScores{p.scores.Block, p.scores.Item, p.scores.Entity} {
		for _, actionScores := range statScores {
			for _, scoreList := range actionScores {
				for _, names := range scoreList {
					for _, name := range names {
						if name == player.Name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
