package cache

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/player"
	"github.com/avernin-one/averninstats/pkg/utils"
	"github.com/rs/zerolog/log"
)

// CachedPlayer holds the resolved name, skin metadata, and cache freshness
// information for a single Minecraft player UUID.
type CachedPlayer struct {
	Name      string      `json:"name"`
	UUID      string      `json:"uuid"`
	LastCheck time.Time   `json:"last_check"`
	SkinURL   string      `json:"skin_url"`
	SkinModel string      `json:"model"`
	Skin      image.Image `json:"-"` // in-memory only, not persisted
}

// PlayerCache manages the on-disk list of resolved players.
type PlayerCache struct {
	Players []*CachedPlayer `json:"players"`

	filePath string
}

var (
	cfg = config.Get()
)

// NewPlayerCache loads the player cache from disk. If the file does not exist
// an empty cache is returned without error.
func NewPlayerCache() *PlayerCache {
	pc := &PlayerCache{
		filePath: filepath.Join(cfg.CacheDir, "playercache.json"),
	}

	data, err := os.ReadFile(pc.filePath)
	if err != nil {
		log.Warn().Str("path", pc.filePath).Msg("player cache not found, starting empty")
		return pc
	}

	if err := json.Unmarshal(data, pc); err != nil {
		log.Error().Err(err).Msg("failed to decode player cache, starting empty")
		return &PlayerCache{filePath: pc.filePath}
	}

	log.Info().Str("path", pc.filePath).Int("players", len(pc.Players)).Msg("player cache loaded")
	return pc
}

// SaveToFile persists the current cache to disk.
func (pc *PlayerCache) SaveToFile() error {
	if err := os.MkdirAll(filepath.Dir(pc.filePath), 0o775); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	data, err := json.MarshalIndent(pc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode player cache: %w", err)
	}

	if err := os.WriteFile(pc.filePath, data, 0o664); err != nil {
		return fmt.Errorf("write player cache: %w", err)
	}

	log.Info().Str("path", pc.filePath).Int("players", len(pc.Players)).Msg("player cache saved")
	return nil
}

// GetOrFetch returns the cached entry for uuid. If the entry is missing,
// expired, or has incomplete skin data it is refreshed from the Mojang API.
func (pc *PlayerCache) GetOrFetch(uuid string) (*CachedPlayer, error) {
	p := pc.findByUUID(uuid)

	if p == nil {
		fetched, err := fetchFromAPI(uuid)
		if err != nil {
			return nil, err
		}
		pc.Players = append(pc.Players, fetched)
		log.Info().Str("name", fetched.Name).Str("uuid", uuid).Msg("new player added to cache")
		return fetched, nil
	}

	if isExpired(p) || p.SkinURL == "" || p.SkinModel == "" {
		if err := refresh(p); err != nil {
			// Non-fatal: return stale entry so processing can continue.
			log.Warn().Err(err).Str("name", p.Name).Msg("failed to refresh player, using stale data")
		}
	}

	return p, nil
}

// GetByUUID returns a player already in the cache without any network
// requests. Returns an error if the UUID is not found.
func (pc *PlayerCache) GetByUUID(uuid string) (*CachedPlayer, error) {
	if p := pc.findByUUID(uuid); p != nil {
		return p, nil
	}
	return nil, fmt.Errorf("player %q not found in cache", uuid)
}

// EnsureSkin loads the skin image into memory if it is not already set,
// then saves head and body renders to disk if they are missing or expired.
func (pc *PlayerCache) EnsureSkin(p *CachedPlayer, outputDir string) {
	if p.Skin == nil {
		p.Skin = player.GetSkin(p.SkinURL)
	}

	if isExpired(p) || !player.HeadExists(outputDir, p.Name) {
		player.SaveHead(p.Skin, outputDir, p.Name, p.SkinModel)
	}

	if isExpired(p) || !player.BodyExists(outputDir, p.Name) {
		player.SaveBody(p.Skin, outputDir, p.Name, p.SkinModel)
	}
}

// --- Internal ----------------------------------------------------------------

func (pc *PlayerCache) findByUUID(uuid string) *CachedPlayer {
	for _, p := range pc.Players {
		if p.UUID == uuid {
			return p
		}
	}
	return nil
}

func isExpired(p *CachedPlayer) bool {
	maxAge := time.Duration(cfg.CacheMaxAge) * time.Hour
	if time.Since(p.LastCheck) > maxAge {
		log.Warn().
			Str("name", p.Name).
			Str("age", time.Since(p.LastCheck).Round(time.Minute).String()).
			Str("max_age", maxAge.String()).
			Msg("player cache entry expired")
		return true
	}
	return false
}

func fetchFromAPI(uuid string) (*CachedPlayer, error) {
	data, err := player.Fetch(uuid)
	if err != nil {
		return nil, fmt.Errorf("fetch player %q from Mojang: %w", uuid, err)
	}

	return &CachedPlayer{
		Name:      data.Name,
		UUID:      uuid,
		LastCheck: utils.AddRandomTime(time.Now(), cfg.LastCheckJitter),
		SkinURL:   data.SkinURL,
		SkinModel: data.SkinModel,
	}, nil
}

func refresh(p *CachedPlayer) error {
	data, err := player.Fetch(p.UUID)
	if err != nil {
		return err
	}

	p.Name = data.Name
	p.SkinURL = data.SkinURL
	p.SkinModel = data.SkinModel
	p.LastCheck = utils.AddRandomTime(time.Now(), cfg.LastCheckJitter)

	log.Info().Str("name", p.Name).Msg("player cache entry refreshed")
	return nil
}
