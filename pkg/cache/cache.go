package cache

import (
	"fmt"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/config"
)

// Returns the asset index file path.
func AssetIndexFile() string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "assetindex.json")
}

// Returns the versioned lookup cache file path.
func LookupFile() string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "lookup.json")
}

// Returns the player cache file path.
func PlayerCacheFile() string {
	return filepath.Join(config.Get().CacheDir, "playercache.json")
}

// Save the raw language file to the cache directory.
func RawLanguageFile(name string) string {
	return filepath.Join(config.Get().CacheDir, config.Get().MinecraftVersion, "lang-raw", fmt.Sprintf("%s.json", name))
}
