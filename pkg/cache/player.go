package cache

import (
	"fmt"
	"image"
	"time"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/player"
	"github.com/avernin-one/averninstats/pkg/utils"
	"github.com/rs/zerolog/log"
)

// CachedPlayer holds the resolved name, skin metadata, and freshness info
// for a single Minecraft player UUID.
type CachedPlayer struct {
	Name      string      `json:"name"`
	UUID      string      `json:"uuid"`
	LastCheck time.Time   `json:"last_check"`
	SkinURL   string      `json:"skin_url"`
	SkinModel string      `json:"model"`
	Skin      image.Image `json:"-"` // in-memory only
}

// PlayerCache manages the persisted list of resolved players.
type PlayerCache []*CachedPlayer

// NewPlayerCache loads the player cache from disk. Returns an empty cache
// (without error) if the file does not exist yet.
func NewPlayerCache() *PlayerCache {
	pc := &PlayerCache{}

	if err := utils.ReadJSONFile(PlayerCacheFile(), pc); err != nil {
		log.Warn().Str("path", PlayerCacheFile()).Msg("player cache not found, starting empty")
		return pc
	}

	log.Info().Str("path", PlayerCacheFile()).Int("players", len(*pc)).Msg("player cache loaded")

	return pc
}

// SaveToFile persists the cache to disk.
func (pc *PlayerCache) Save() {
	if err := utils.SaveJSONFile(PlayerCacheFile(), pc); err != nil {
		log.Error().Err(err).Str("path", PlayerCacheFile()).Msg("failed to save player cache")
	}
}

// GetOrFetch returns the cached entry for uuid, fetching from the Mojang API
// if the entry is missing, expired, or has incomplete skin data.
func (pc *PlayerCache) GetOrFetch(uuid string) (*CachedPlayer, error) {
	cp := pc.GetByUUID(uuid)

	if cp == nil {
		fetched, err := pc.fetchFromAPI(uuid)
		if err != nil {
			return nil, err
		}

		*pc = append(*pc, fetched)

		log.Info().Str("name", fetched.Name).Str("uuid", uuid).Msg("new player added to cache")

		return fetched, nil
	}

	cp.EnsureSkin()

	if cp.IsExpired() {
		if err := cp.Refresh(); err != nil {
			log.Warn().Err(err).Str("name", cp.Name).Msg("failed to refresh player, using stale data")
		}
	}

	return cp, nil
}

// GetByUUID returns a cached player without any network requests.
// Returns an error if the UUID is not found.
func (pc *PlayerCache) GetByUUID(uuid string) *CachedPlayer {
	for _, p := range *pc {
		if p.UUID == uuid {
			return p
		}
	}

	return nil
}

// EnsureSkin downloads the skin image if not already in memory, then renders
// and saves head/body images to disk if they are missing or expired.
func (cp *CachedPlayer) EnsureSkin() {
	if cp.Skin == nil {
		cp.Skin = player.GetSkin(cp.SkinURL)
	}

	if !player.HeadExists(config.Get().OutputDir, cp.Name) {
		player.SaveHead(cp.Skin, config.Get().OutputDir, cp.Name, cp.SkinModel)
	}

	if !player.BodyExists(config.Get().OutputDir, cp.Name) {
		player.SaveBody(cp.Skin, config.Get().OutputDir, cp.Name, cp.SkinModel)
	}
}

func (cp *CachedPlayer) IsExpired() bool {
	maxAge := time.Duration(config.Get().CacheMaxAge) * time.Hour
	if time.Since(cp.LastCheck) > maxAge {
		log.Warn().
			Str("name", cp.Name).
			Str("age", time.Since(cp.LastCheck).Round(time.Minute).String()).
			Str("max_age", maxAge.String()).
			Msg("player cache entry expired")
		return true
	}

	return false
}

func (pc *PlayerCache) fetchFromAPI(uuid string) (*CachedPlayer, error) {
	data, err := player.Fetch(uuid, config.Get().QueryDelay)
	if err != nil {
		return nil, fmt.Errorf("fetch player %q: %w", uuid, err)
	}

	return &CachedPlayer{
		Name:      data.Name,
		UUID:      uuid,
		LastCheck: utils.AddRandomTime(time.Now(), config.Get().LastCheckJitter),
		SkinURL:   data.SkinURL,
		SkinModel: data.SkinModel,
	}, nil
}

func (cp *CachedPlayer) Refresh() error {
	data, err := player.Fetch(cp.UUID, config.Get().QueryDelay)
	if err != nil {
		return err
	}

	cp.Name = data.Name
	cp.SkinURL = data.SkinURL
	cp.SkinModel = data.SkinModel
	cp.LastCheck = utils.AddRandomTime(time.Now(), config.Get().LastCheckJitter)

	cp.EnsureSkin()

	log.Info().Str("name", cp.Name).Msg("player cache entry refreshed")

	return nil
}
