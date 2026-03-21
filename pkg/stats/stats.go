package stats

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

// --- Score types -------------------------------------------------------------

// ScoreList maps a numeric score to the player names that achieved it.
type ScoreList map[int][]string

// ActionScores maps an action type (mined, crafted, …) to its score list.
type ActionScores map[string]ScoreList

// StatScores maps a stat name (stone, creeper, …) to its per-action scores.
type StatScores map[string]ActionScores

// --- Per-player types --------------------------------------------------------

// PlayerScores holds a player's top scores per category (block/item/entity).
// Structure: category → action → score → []statName
type PlayerScores map[string]map[string]ScoreList

// Medals counts a player's global highscore placements.
type Medals struct {
	Gold   int `json:"gold"`
	Silver int `json:"silver"`
	Bronze int `json:"bronze"`
}

// Player aggregates all processed data for a single Minecraft player.
type Player struct {
	Name   string         `json:"name"`
	UUID   string         `json:"uuid"`
	Medals Medals         `json:"medals"`
	Stats  map[string]int `json:"stats"`  // flattened minecraft:custom values
	Scores PlayerScores   `json:"scores"` // top-N per category/action
}

// --- Global output types -----------------------------------------------------

// HighscoreEntry is written per minecraft:custom stat.
type HighscoreEntry struct {
	Name   string    `json:"name"`
	Scores ScoreList `json:"scores"`
}

// StatEntry is written per block / item / entity stat.
type StatEntry struct {
	Name   string       `json:"name"`
	Type   string       `json:"type"` // "block" | "item" | "entity"
	Scores ActionScores `json:"scores"`
}

// --- Processor ---------------------------------------------------------------

type Processor struct {
	cfg        *config.Config
	lookup     *cache.Lookup
	highscores map[string]ScoreList
	scores     struct {
		Block  StatScores
		Item   StatScores
		Entity StatScores
	}
	players []*Player
}

func New(l *cache.Lookup) *Processor {
	p := &Processor{
		cfg:        config.Get(),
		lookup:     l,
		highscores: make(map[string]ScoreList),
	}
	p.scores.Block = make(StatScores)
	p.scores.Item = make(StatScores)
	p.scores.Entity = make(StatScores)
	return p
}

// ProcessFile reads one player stats JSON file and integrates it into the
// global aggregates. uuid is the filename without extension.
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

// Flush writes all aggregated JSON output files and returns the player list.
func (p *Processor) Flush() ([]*Player, error) {
	if err := p.writeHighscores(); err != nil {
		return nil, err
	}
	if err := p.writeStatFiles(cache.TypeBlock, p.scores.Block); err != nil {
		return nil, err
	}
	if err := p.writeStatFiles(cache.TypeItem, p.scores.Item); err != nil {
		return nil, err
	}
	if err := p.writeStatFiles(cache.TypeEntity, p.scores.Entity); err != nil {
		return nil, err
	}
	return p.players, nil
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
		trimScoreList(p.highscores[stat], p.cfg.NumHighscores)
	}
}

func (p *Processor) processStatEntry(player *Player, action, stat string, count int) {
	type target struct {
		lookupList []string
		scores     *StatScores
		category   string
	}

	targets := []target{
		{p.lookup.Block, &p.scores.Block, cache.TypeBlock},
		{p.lookup.Item, &p.scores.Item, cache.TypeItem},
		{p.lookup.Entity, &p.scores.Entity, cache.TypeEntity},
	}

	for _, t := range targets {
		if t.category == cache.TypeItem && action == "killed" {
			continue
		}
		if !slices.Contains(t.lookupList, stat) {
			continue
		}

		// Global stat scores
		if (*t.scores)[stat] == nil {
			(*t.scores)[stat] = make(ActionScores)
		}
		if (*t.scores)[stat][action] == nil {
			(*t.scores)[stat][action] = make(ScoreList)
		}
		(*t.scores)[stat][action][count] = append((*t.scores)[stat][action][count], player.UUID)
		trimScoreList((*t.scores)[stat][action], p.cfg.NumHighscores)

		// Player personal top-N
		playerCat := player.Scores[t.category]
		if playerCat[action] == nil {
			playerCat[action] = make(ScoreList)
		}
		playerCat[action][count] = append(playerCat[action][count], stat)
		trimScoreList(playerCat[action], p.cfg.NumPlayerHighscores)
	}
}

// --- Output ------------------------------------------------------------------

func (p *Processor) writeHighscores() error {
	for name, scores := range p.highscores {
		path := filepath.Join(p.cfg.OutputDir, cache.TypeHighscore, fmt.Sprintf("%s.json", name))
		if err := saveJSON(path, HighscoreEntry{Name: name, Scores: scores}); err != nil {
			log.Warn().Err(err).Str("stat", name).Msg("failed to write highscore file")
		}
	}
	return nil
}

func (p *Processor) writeStatFiles(category string, data StatScores) error {
	for name, actionScores := range data {
		path := filepath.Join(p.cfg.OutputDir, category, fmt.Sprintf("%s.json", name))
		if err := saveJSON(path, StatEntry{Name: name, Type: category, Scores: actionScores}); err != nil {
			log.Warn().Err(err).Str("category", category).Str("stat", name).Msg("failed to write stat file")
		}
	}
	return nil
}

// --- Helpers -----------------------------------------------------------------

func trimNamespace(key string) string {
	if i := strings.IndexByte(key, ':'); i >= 0 {
		return key[i+1:]
	}
	log.Warn().Str("key", key).Msg("stats key missing namespace, using as-is")
	return key
}

func trimScoreList(list ScoreList, max int) {
	keys := sortedKeys(list)
	if len(keys) <= max {
		return
	}
	for _, k := range keys[max:] {
		delete(list, k)
	}
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
	log.Debug().Str("path", path).Msg("saved")
	return nil
}
