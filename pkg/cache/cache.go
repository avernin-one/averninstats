package cache

import (
	"fmt"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/config"
)

// @TODO
func AssetIndexFile() string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "assetindex.json")
}

// LookupCachePath returns the versioned lookup cache file path.
func LookupFile() string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "lookup.json")
}

// PlayerCachePath returns the player cache file path.
func PlayerCacheFile() string {
	return filepath.Join(config.Get().CacheDir, "playercache.json")
}

// @TODO
func RawLanguageFile(name string) string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "lang-raw", fmt.Sprintf("%s.json", name))
}
