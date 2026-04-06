package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
)

// This is the format of the "stats" subkey of every player json file
type Stats map[string]map[string]int

// Counts first, second, and third place highscore positions.
type Medals struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Bronze int `json:"bronze"`
}

// JSON output for a single player file.
type Player struct {
	Name   string         `json:"name"`
	UUID   string         `json:"uuid"`
	Medals Medals         `json:"medals"`
	Stats  map[string]int `json:"stats"` // minecraft:custom values
	//        category   action     score[]statName.
	//        block      dropped    5    stone,oak_log
	Scores map[string]map[string]map[int][]string `json:"scores"`
}

// Is written per minecraft:custom stat.
type HighscoreEntry struct {
	Name   string           `json:"name"`
	Scores map[int][]string `json:"scores"`
}

// Is written per block/item/entity stat.
type StatEntry struct {
	Name   string                                 `json:"name"`
	Scores map[string]map[string]map[int][]string `json:"scores"`
}

type Processor struct {
	Lookup      *cache.Lookup
	PlayerCache *cache.PlayerCache
	Highscores  map[string]map[int][]string
	//         block       stone      mined     5    player1,player2
	Scores      map[string]map[string]map[string]map[int][]string
	PlayerNames []string
}

func New(lookup *cache.Lookup, pc *cache.PlayerCache) *Processor {
	p := &Processor{
		Lookup:      lookup,
		PlayerCache: pc,
		Highscores:  map[string]map[int][]string{},
		Scores:      map[string]map[string]map[string]map[int][]string{},
	}

	return p
}

func (p *Processor) Process() {
	files, err := os.ReadDir(config.Get().StatsSourceDir)
	if err != nil {
		log.Fatal().Err(err).Str("dir", config.Get().StatsSourceDir).Msg("cannot read stats source directory")
	}

	current := 0
	skipped := 0
	total := len(files)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			skipped++
			log.Info().Bool("is_dir", file.IsDir()).Str("name", file.Name()).Msg("skipping non-JSON file")
			continue
		}

		current++
		uuid := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		path := filepath.Join(config.Get().StatsSourceDir, file.Name())

		log.Info().
			Str("file", file.Name()).
			Str("progress", fmt.Sprintf("%d/%d/%d", current, skipped, total)).
			Msg("processing stats file")

		var statsFile struct {
			Stats Stats `json:"stats"`
		}

		data, err := os.ReadFile(path)
		if err != nil {
			log.Error().Err(err).Str("uuid", uuid).Msg("unable to read file")
			continue
		}

		if err := json.Unmarshal(data, &statsFile); err != nil {
			log.Error().Err(err).Str("uuid", uuid).Msg("unable to unmarshal data")
			continue
		}

		p.processStats(statsFile.Stats, uuid)
	}

	p.WriteIndexes()
	p.WriteStats()
	p.WriteHighscore()
}

func (p *Processor) processStats(rawStats Stats, uuid string) {
	if !p.hasMinPlaytime(rawStats) {
		log.Warn().Str("uuid", uuid).Msg("not enough playtime")
		return
	}

	cachedPlayer, err := p.PlayerCache.GetOrFetch(uuid)
	if err != nil {
		log.Error().Err(err).Msg("unable to get player name")
		return
	}

	player := &Player{
		UUID:   uuid,
		Name:   cachedPlayer.Name,
		Scores: make(map[string]map[string]map[int][]string),
		Medals: Medals{},
	}

	// action is one of
	//	- custom
	//	- picked_up
	//	- mined
	//	- killed_by
	//	- broken
	//	- killed
	//	- used
	//	- dropped
	//	- crafted
	for action, entries := range rawStats {
		action = trimNamespace(action)

		if action == cache.TypeCustom {
			p.processHighscore(entries, player)
			continue
		}

		for stat, count := range entries {
			p.processStatEntry(player, action, trimNamespace(stat), count)
		}
	}

	p.setMedals(player)
	p.WritePlayer(player)

	p.PlayerNames = append(p.PlayerNames, player.Name)
}

func (p *Processor) hasMinPlaytime(rawStats Stats) bool {
	if _, exists := rawStats["minecraft:custom"]; !exists {
		return false
	}

	ticks, exists := rawStats["minecraft:custom"]["minecraft:play_time"]
	if !exists {
		return false
	}

	return ticks/20/60 > config.Get().MinPlayTime
}

// adds the custom stats to each player and also fills the highscores
// for each statistic.
func (p *Processor) processHighscore(entries map[string]int, player *Player) {
	player.Stats = make(map[string]int, len(entries))

	for stat, count := range entries {
		stat := trimNamespace(stat)
		player.Stats[stat] = count

		if p.Highscores[stat] == nil {
			p.Highscores[stat] = make(map[int][]string)
		}

		p.Highscores[stat][count] = append(p.Highscores[stat][count], player.Name)
		trimScoreList(p.Highscores[stat], config.Get().NumHighscores)
	}
}

func (p *Processor) processStatEntry(player *Player, action, stat string, count int) {
	category, err := p.Lookup.GetType(stat)
	if err != nil {
		// This is actually ok, if it does not exists in the LookupMap no one has
		// ever used a block or entity so we skip this one.
		log.Debug().Err(err).Str("stat", stat).Str("action", action).Msg("not found")
		return
	}

	if category == cache.TypeItem && action == "killed" {
		return
	}

	if category == cache.TypeBlock && stat == "air" {
		return
	}

	// Create player scores.
	if (player.Scores)[category] == nil {
		(player.Scores)[category] = make(map[string]map[int][]string)
	}

	if (player.Scores)[category][action] == nil {
		(player.Scores)[category][action] = make(map[int][]string)
	}

	(player.Scores)[category][action][count] = append((player.Scores)[category][action][count], stat)
	trimScoreList((player.Scores)[category][action], config.Get().NumPlayerHighscores)
	slices.Sort((player.Scores)[category][action][count])

	// Create global scores.
	if (p.Scores)[category] == nil {
		(p.Scores)[category] = make(map[string]map[string]map[int][]string)
	}

	if (p.Scores)[category][stat] == nil {
		(p.Scores)[category][stat] = make(map[string]map[int][]string)
	}

	if (p.Scores)[category][stat][action] == nil {
		(p.Scores)[category][stat][action] = make(map[int][]string)
	}

	(p.Scores)[category][stat][action][count] = append((p.Scores)[category][stat][action][count], player.Name)
	trimScoreList((p.Scores)[category][stat][action], config.Get().NumHighscores)
	slices.Sort((p.Scores)[category][stat][action][count])
}
