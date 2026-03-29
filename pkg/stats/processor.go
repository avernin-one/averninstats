package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
)

// ScoreList maps a score value to the players that achieved it.
type ScoreList map[int][]string

// ActionScores maps an action (mined, crafted, ...) to its score list.
type ActionScores map[string]ScoreList

// StatScores maps a stat name (stone, creeper, ...) to its per-action scores.
type StatScores map[string]ActionScores

// PlayerScores holds a player's personal top-N scores per category.
// Structure: category -> action -> score -> []statName
type PlayerScores map[string]map[string]ScoreList

// Medals counts first, second, and third place highscore positions.
type Medals struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Bronze int `json:"bronze"`
}

// Player is the JSON output for a single player file.
type Player struct {
	Name   string         `json:"name"`
	UUID   string         `json:"uuid"`
	Medals Medals         `json:"medals"`
	Stats  map[string]int `json:"stats"`  // minecraft:custom values
	Scores PlayerScores   `json:"scores"` // personal top-N
}

// HighscoreEntry is written per minecraft:custom stat.
type HighscoreEntry struct {
	Name   string    `json:"name"`
	Scores ScoreList `json:"scores"`
}

// StatEntry is written per block/item/entity stat.
type StatEntry struct {
	Name   string       `json:"name"`
	Type   string       `json:"type"`
	Scores ActionScores `json:"scores"`
}

// --- Processor ---------------------------------------------------------------

// Processor aggregates per-player stats files into global highscores and
// per-stat score lists. UUIDs are stored internally during ProcessFile and
// replaced with real names in WritePlayerFiles.
type Processor struct {
	lookup     *cache.Lookup
	highscores map[string]ScoreList
	scores     struct {
		Block  StatScores
		Item   StatScores
		Entity StatScores
	}
	players []*Player
	// rankedUUIDs tracks which UUIDs appear in any score list, enabling
	// playerIsRanked checks without re-traversing all score maps.
	rankedUUIDs map[string]struct{}
}

func New(l *cache.Lookup) *Processor {
	p := &Processor{
		lookup:      l,
		highscores:  make(map[string]ScoreList),
		rankedUUIDs: make(map[string]struct{}),
	}

	p.scores.Block = make(StatScores)
	p.scores.Item = make(StatScores)
	p.scores.Entity = make(StatScores)
	return p
}

// ProcessFile reads one player stats JSON file and integrates it into the
// aggregates. uuid is the filename without its .json extension.
func (p *Processor) ProcessFile(filePath, uuid string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", filePath, err)
	}

	var root struct {
		Stats map[string]map[string]int `json:"stats"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("decode %q: %w", filePath, err)
	}

	p.processPlayer(root.Stats, uuid)
	return nil
}

// Flush resolves UUIDs to player names using the provided cache, writes all
// player JSON files, and then writes all global stat/highscore files.
// The player cache is the only source of truth for player names.
func (p *Processor) Flush(pc *cache.PlayerCache) error {
	// Build uuid -> name map in one pass.
	uuidToName := make(map[string]string, len(p.players))
	for _, player := range p.players {
		cp, err := pc.GetOrFetch(player.UUID)
		if err != nil {
			log.Warn().Err(err).Str("uuid", player.UUID).Msg("failed to resolve player, skipping")
			continue
		}
		uuidToName[player.UUID] = cp.Name
		player.Name = cp.Name
	}

	// Resolve UUIDs -> names in all score maps.
	p.resolveNames(uuidToName)

	// Now write player files (medals depend on resolved names in highscores).
	for _, player := range p.players {
		if player.Name == "" {
			continue // resolution failed above
		}
		p.setMedals(player)

		if !p.meetsPlaytimeRequirement(player) {
			log.Warn().Str("name", player.Name).Int("min_playtime", config.Get().MinPlayTime).Msg("player below minimum playtime, skipping")
			continue
		}

		path := filepath.Join(config.Get().OutputDir, cache.TypePlayer,
			fmt.Sprintf("%s.json", player.Name))
		if err := saveJSON(path, player); err != nil {
			log.Warn().Err(err).Str("player", player.Name).Msg("failed to write player file")
		}

		cp, err := pc.GetByUUID(player.UUID)
		if err == nil {
			pc.EnsureSkin(cp)
		}
	}

	// Write global stat files.
	for name, scores := range p.highscores {
		path := filepath.Join(config.Get().OutputDir, cache.TypeHighscore,
			fmt.Sprintf("%s.json", name))
		if err := saveJSON(path, HighscoreEntry{Name: name, Scores: scores}); err != nil {
			log.Warn().Err(err).Str("stat", name).Msg("failed to write highscore file")
		}
	}

	for _, group := range []struct {
		typ  string
		data StatScores
	}{
		{cache.TypeBlock, p.scores.Block},
		{cache.TypeItem, p.scores.Item},
		{cache.TypeEntity, p.scores.Entity},
	} {
		for name, actionScores := range group.data {
			path := filepath.Join(config.Get().OutputDir, group.typ,
				fmt.Sprintf("%s.json", name))
			if err := saveJSON(path, StatEntry{Name: name, Type: group.typ, Scores: actionScores}); err != nil {
				log.Warn().Err(err).Str("type", group.typ).Str("stat", name).Msg("failed to write stat file")
			}
		}
	}

	return nil
}

// --- Internal processing -----------------------------------------------------

func (p *Processor) processPlayer(raw map[string]map[string]int, uuid string) {
	player := &Player{
		UUID: uuid,
		Scores: PlayerScores{
			cache.TypeBlock:  make(map[string]ScoreList),
			cache.TypeItem:   make(map[string]ScoreList),
			cache.TypeEntity: make(map[string]ScoreList),
		},
	}

	for fullAction, entries := range raw {
		action := trimNamespace(fullAction)

		if action == cache.TypeCustom {
			p.processCustom(entries, player)
			continue
		}
		for fullStat, count := range entries {
			p.processStatEntry(player, action, trimNamespace(fullStat), count)
		}
	}

	p.players = append(p.players, player)
}

func (p *Processor) processCustom(entries map[string]int, player *Player) {
	player.Stats = make(map[string]int, len(entries))

	for fullStat, count := range entries {
		stat := trimNamespace(fullStat)
		player.Stats[stat] = count

		if p.highscores[stat] == nil {
			p.highscores[stat] = make(ScoreList)
		}
		p.highscores[stat][count] = append(p.highscores[stat][count], player.UUID)
		trimScoreList(p.highscores[stat], config.Get().NumHighscores)
	}
}

func (p *Processor) processStatEntry(player *Player, action, stat string, count int) {
	type target struct {
		contains func(string) bool
		scores   *StatScores
		category string
	}

	// Set lookups
	targets := []target{
		{p.lookup.ContainsBlock, &p.scores.Block, cache.TypeBlock},
		{p.lookup.ContainsItem, &p.scores.Item, cache.TypeItem},
		{p.lookup.ContainsEntity, &p.scores.Entity, cache.TypeEntity},
	}

	for _, t := range targets {
		if t.category == cache.TypeItem && action == "killed" {
			continue
		}
		if !t.contains(stat) {
			continue
		}

		// Global scores
		if (*t.scores)[stat] == nil {
			(*t.scores)[stat] = make(ActionScores)
		}
		if (*t.scores)[stat][action] == nil {
			(*t.scores)[stat][action] = make(ScoreList)
		}
		(*t.scores)[stat][action][count] = append((*t.scores)[stat][action][count], player.UUID)
		trimScoreList((*t.scores)[stat][action], config.Get().NumHighscores)

		// Player personal top-N
		playerCat := player.Scores[t.category]
		if playerCat[action] == nil {
			playerCat[action] = make(ScoreList)
		}
		playerCat[action][count] = append(playerCat[action][count], stat)
		trimScoreList(playerCat[action], config.Get().NumPlayerHighscores)

		// Track that this UUID appears in at least one ranking.
		p.rankedUUIDs[player.UUID] = struct{}{}
	}
}

// resolveNames replaces all UUID placeholders with real names
func (p *Processor) resolveNames(uuidToName map[string]string) {
	replace := func(list ScoreList) {
		for score, entries := range list {
			for i, v := range entries {
				if name, ok := uuidToName[v]; ok {
					list[score][i] = name
				}
			}
		}
	}

	for _, sl := range p.highscores {
		replace(sl)
	}
	for _, statScores := range []StatScores{p.scores.Block, p.scores.Item, p.scores.Entity} {
		for _, actionScores := range statScores {
			for _, sl := range actionScores {
				replace(sl)
			}
		}
	}
}

// setMedals counts top-1/2/3 positions in the global highscores for player.
func (p *Processor) setMedals(player *Player) {
	player.Medals = Medals{}
	for _, scoreList := range p.highscores {
		for rank, key := range sortedKeys(scoreList) {
			if rank >= 3 {
				break // only gold/silver/bronze
			}

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
// or already appears in any score ranking.
func (p *Processor) meetsPlaytimeRequirement(player *Player) bool {
	if _, ranked := p.rankedUUIDs[player.UUID]; ranked {
		return true
	}

	for _, sl := range p.highscores {
		for _, names := range sl {
			for _, name := range names {
				if name == player.Name {
					return true
				}
			}
		}
	}

	ticks, ok := player.Stats["play_time"]
	if !ok {
		return false
	}

	return ticks/20/60 >= config.Get().MinPlayTime
}

// --- Helpers -----------------------------------------------------------------

func trimNamespace(key string) string {
	if i := strings.IndexByte(key, ':'); i >= 0 {
		return key[i+1:]
	}

	log.Warn().Str("key", key).Msg("stats key missing namespace separator")

	return key
}

func saveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	if err := os.WriteFile(path, data, 0o664); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	log.Debug().Str("path", path).Msg("file saved")

	return nil
}
