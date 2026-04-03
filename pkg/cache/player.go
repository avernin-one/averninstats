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
	p, err := pc.GetByUUID(uuid)
	if err != nil {
		return nil, err
	}

	if p == nil {
		fetched, err := pc.fetchFromAPI(uuid)
		if err != nil {
			return nil, err
		}

		*pc = append(*pc, fetched)

		log.Info().Str("name", fetched.Name).Str("uuid", uuid).Msg("new player added to cache")

		return fetched, nil
	}

	if pc.isExpired(p) || p.SkinURL == "" || p.SkinModel == "" {
		if err := pc.refresh(p); err != nil {
			log.Warn().Err(err).Str("name", p.Name).Msg("failed to refresh player, using stale data")
		}
	}

	return p, nil
}

// GetByUUID returns a cached player without any network requests.
// Returns an error if the UUID is not found.
func (pc *PlayerCache) GetByUUID(uuid string) (*CachedPlayer, error) {
	for _, p := range *pc {
		if p.UUID == uuid {
			return p, nil
		}
	}
	return nil, fmt.Errorf("player %q not found in cache", uuid)
}

// EnsureSkin downloads the skin image if not already in memory, then renders
// and saves head/body images to disk if they are missing or expired.
func (pc *PlayerCache) EnsureSkin(p *CachedPlayer) {
	if p.Skin == nil {
		p.Skin = player.GetSkin(p.SkinURL)
	}
	if pc.isExpired(p) || !player.HeadExists(config.Get().OutputDir, p.Name) {
		player.SaveHead(p.Skin, config.Get().OutputDir, p.Name, p.SkinModel)
	}
	if pc.isExpired(p) || !player.BodyExists(config.Get().OutputDir, p.Name) {
		player.SaveBody(p.Skin, config.Get().OutputDir, p.Name, p.SkinModel)
	}
}

func (pc *PlayerCache) isExpired(p *CachedPlayer) bool {
	maxAge := time.Duration(config.Get().CacheMaxAge) * time.Hour
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

func (pc *PlayerCache) refresh(p *CachedPlayer) error {
	data, err := player.Fetch(p.UUID, config.Get().QueryDelay)
	if err != nil {
		return err
	}
	p.Name = data.Name
	p.SkinURL = data.SkinURL
	p.SkinModel = data.SkinModel
	p.LastCheck = utils.AddRandomTime(time.Now(), config.Get().LastCheckJitter)
	log.Info().Str("name", p.Name).Msg("player cache entry refreshed")
	return nil
}
