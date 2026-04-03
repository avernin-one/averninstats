package translations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/avernin-one/averninstats/pkg/cache"
	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/utils"
	"github.com/rs/zerolog/log"
)

type versionManifest struct {
	Versions []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"versions"`
}

type versionData struct {
	AssetIndex struct {
		URL string `json:"url"`
	} `json:"assetIndex"` //nolint:tagliatelle // defined by mojoang
}

type assetIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int    `json:"size"`
	} `json:"objects"`
}

var (
	// populateRe classifies a raw Mojang key into block/item/entity/stat.
	populateRe = regexp.MustCompile(`^(block|item|entity|stat)(\.minecraft)?\.`)
	// processRe strips the full namespaced prefix.
	processRe = regexp.MustCompile(`^(block|item|entity|stats?(_type)?)(\.minecraft)?\.`)
	// lookupSource is the language used to build the Lookup.
	lookupSource = "en-gb"
)

// Run writes processed i18n files to config.Get().I18nDir() and returns a
// ready-to-use Lookup.
//
// The raw language files are cached under <cacheDir>/<version>/lang-raw so
// subsequent runs skip downloading. Processed i18n files are currently always
// re-written so version changes are always picked up.
//
// The Lookup itself is cached separately in lookup.json. If that cache is
// valid the download/processing step is still run for the i18n files, but
// the Lookup is returned from cache instead of being rebuilt.
func Run() (*cache.Lookup, error) {
	index, err := fetchAssetIndex(config.Get().MinecraftVersion)
	if err != nil {
		// If the asset index is unavailable:
		// - try returning a cached lookup
		// - unless any cached lookup is empty, in which case return the error
		if l, cacheErr := cache.LoadLookup(); cacheErr == nil && !l.AnyEmpty() {
			log.Warn().Err(err).Msg("asset index unavailable, using cached lookup")
			return l, nil
		}

		return nil, err
	}

	log.Info().Str("version", config.Get().MinecraftVersion).Msg("processing language files")

	l := &cache.Lookup{}
	const langPrefix string = "minecraft/lang/"

	var languages []string

	for key, obj := range index.Objects {
		processLanguage(l, &languages, key, obj.Hash, langPrefix)
	}

	if l.AnyEmpty() {
		return nil, fmt.Errorf(
			"lookup source language %q not found in asset index, check --minecraft-version",
			lookupSource,
		)
	}

	sort.Strings(l.Block)
	sort.Strings(l.Item)
	sort.Strings(l.Entity)
	sort.Strings(l.Custom)
	sort.Strings(languages)

	if err := l.Save(); err != nil {
		log.Error().Err(err).Msg("failed to persist lookup cache")
	}

	if err := writeManifest(languages); err != nil {
		log.Error().Err(err).Msg("failed to write i18n manifest")
	}

	return l, nil
}

func processLanguage(l *cache.Lookup, languages *[]string, key, hash, langPrefix string) {
	if !strings.HasPrefix(key, langPrefix) {
		return
	}

	name := langName(key, langPrefix)

	// Fetch from raw cache or download.
	raw, err := getRaw(name, hash)
	if err != nil {
		log.Error().Err(err).Str("language", name).Msg("failed to get language, skipping")
		return
	}

	// Build the lookup from the source language.
	if name == lookupSource {
		populateLookup(l, raw)
	}

	// Always re-write the processed i18n file so version changes are picked up.
	if err := writeProcessed(name, raw); err != nil {
		log.Warn().Err(err).Str("language", name).Msg("failed to write processed language file")
	}

	*languages = append(*languages, name)
}

// getRaw returns the raw language map for a given language. It first checks
// the local raw cache; if not found it downloads from Mojang and caches it.
func getRaw(name, hash string) (map[string]string, error) {
	rawPath := cache.RawLanguageFile(name)

	if utils.FileExists(rawPath, true) {
		raw, err := loadRaw(rawPath)
		if err == nil {
			log.Debug().Str("language", name).Msg("raw language loaded from cache")
			return raw, nil
		}

		log.Warn().Err(err).Str("language", name).Msg("raw cache unreadable, re-downloading")
	}

	return downloadRaw(name, hash)
}

// downloadRaw fetches a raw language file from Mojang's resource CDN and
// persists it to the raw cache.
func downloadRaw(name, hash string) (map[string]string, error) {
	url := fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", hash[:2], hash)

	body, err := utils.NewHttpRequest(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}

	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}

	if err := utils.SaveJSONFile(cache.RawLanguageFile(name), raw); err != nil {
		log.Warn().Err(err).Str("language", name).Msg("failed to cache raw language file")
	}

	log.Debug().Str("language", name).Str("url", url).Msg("language downloaded")

	return raw, nil
}

// writeProcessed strips irrelevant keys and writes the processed language
// file to config.Get().I18nDir(). It always overwrites any existing file.
func writeProcessed(name string, raw map[string]string) error {
	outPath := filepath.Join(config.Get().I18nDir(), fmt.Sprintf("%s.json", name))
	processed := stripTranslations(raw)

	if err := utils.SaveJSONFile(outPath, processed); err != nil {
		return fmt.Errorf("save %s: %w", name, err)
	}

	return nil
}

// writeManifest writes i18n/_manifest.json listing all available languages.
func writeManifest(languages []string) error {
	path := filepath.Join(config.Get().I18nDir(), "_manifest.json")
	return utils.SaveJSONFile(path, languages)
}

// ---------------------------------------------------------------------------
// Mojang API
// ---------------------------------------------------------------------------

func fetchAssetIndex(minecraftVersion string) (*assetIndex, error) {
	var index assetIndex

	if utils.FileExists(cache.AssetIndexFile(), true) {
		if err := utils.ReadJSONFile(cache.AssetIndexFile(), &index); err == nil {
			log.Info().Str("cache", minecraftVersion).Msg("asset index loaded from cache")
			return &index, nil
		}
	}

	// If the asset index is unavailable, the program can still run using cached
	// lookups and previously processed i18n files, so this is a warning, not an error.
	log.Warn().Str("version", minecraftVersion).Msg("asset index not found in cache, downloading")

	body, err := utils.NewHttpRequest("https://piston-meta.mojang.com/mc/game/version_manifest_v2.json")
	if err != nil {
		return nil, fmt.Errorf("fetch version manifest: %w", err)
	}

	var manifest versionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("decode version manifest: %w", err)
	}

	var versionURL string
	for _, v := range manifest.Versions {
		if v.ID == minecraftVersion {
			versionURL = v.URL
			break
		}
	}

	if versionURL == "" {
		return nil, fmt.Errorf("version %q not found in manifest", minecraftVersion)
	}

	log.Info().Str("version", minecraftVersion).Str("url", versionURL).Msg("version URL resolved")

	body, err = utils.NewHttpRequest(versionURL)
	if err != nil {
		return nil, fmt.Errorf("fetch version data: %w", err)
	}

	var ver versionData
	if err := json.Unmarshal(body, &ver); err != nil {
		return nil, fmt.Errorf("decode version data: %w", err)
	}

	if ver.AssetIndex.URL == "" {
		return nil, fmt.Errorf("asset index URL missing for version %q", minecraftVersion)
	}

	log.Info().Str("url", ver.AssetIndex.URL).Msg("asset index URL resolved")

	body, err = utils.NewHttpRequest(ver.AssetIndex.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch asset index: %w", err)
	}

	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("decode asset index: %w", err)
	}

	if err := utils.SaveJSONFile(cache.AssetIndexFile(), index); err != nil {
		log.Warn().Err(err).Str("version", minecraftVersion).Msg("failed to cache asset index")
		return &index, err
	}

	return &index, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// langName converts an asset index key like "minecraft/lang/en_gb.json" to "en-gb".
func langName(key, prefix string) string {
	name := strings.TrimPrefix(key, prefix)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return name
}

// loadRaw reads and decodes a raw language file from disk.
func loadRaw(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// stripTranslations keeps only block/item/entity/stat keys and strips their
// namespace prefix, cleaning up placeholder artifacts in the values.
func stripTranslations(raw map[string]string) map[string]string {
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if !processRe.MatchString(key) {
			continue
		}

		stripped := processRe.ReplaceAllString(key, "")
		cleaned := strings.TrimSpace(
			strings.ReplaceAll(strings.ReplaceAll(val, "%s", ""), "  ", " "),
		)

		out[stripped] = cleaned
	}

	return out
}

// populateLookup fills a Lookup from the source language raw map.
func populateLookup(l *cache.Lookup, raw map[string]string) {
	for key := range raw {
		match := populateRe.FindStringSubmatch(key)
		if len(match) < 2 {
			continue
		}

		stripped := processRe.ReplaceAllString(key, "")
		if stripped == "air" || stripped == "cave_air" || stripped == "void_air" {
			continue
		}

		switch match[1] {
		case "block":
			l.Block = append(l.Block, stripped)
		case "item":
			l.Item = append(l.Item, stripped)
		case "entity":
			l.Entity = append(l.Entity, stripped)
		case "stat":
			l.Custom = append(l.Custom, stripped)
		default:
			log.Warn().Str("prefix", match[1]).Str("key", key).Msg("unknown key prefix")
		}
	}
}
