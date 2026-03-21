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
	} `json:"assetIndex"`
}

type assetIndex struct {
	Objects map[string]struct {
		Hash string `json:"hash"`
		Size int    `json:"size"`
	} `json:"objects"`
}

var (
	cfg = config.Get()

	// populateRe classifies a raw Mojang key into block / item / entity / stat.
	populateRe = regexp.MustCompile(`^(block|item|entity|stat)(\.minecraft)?\.`)
	// processRe strips the full namespaced prefix to produce the short key stored on disk.
	processRe = regexp.MustCompile(`^(block|item|entity|stats?(_type)?)(\.minecraft)?\.`)

	// lookupSource is the language whose raw keys are used to build the Lookup.
	// en-gb covers all vanilla categories.
	lookupSource = "en-gb"
)

// Run returns a ready-to-use Lookup. If a complete lookup already exists in
// the cache it is returned without any network traffic. Otherwise all language
// files for the configured Minecraft version are downloaded, the lookup is
// built from the en-gb keys, and both are persisted to cache.
func Run() (*cache.Lookup, error) {
	if l, err := cache.Load(); err == nil && !l.AnyEmpty() {
		log.Info().Msg("lookup loaded from cache, skipping download")
		return l, nil
	}

	log.Info().Str("version", cfg.MinecraftVersion).Msg("building lookup from language files")

	index, err := fetchAssetIndex()
	if err != nil {
		return nil, err
	}

	l := &cache.Lookup{}
	const langPrefix = "minecraft/lang/"

	for key, obj := range index.Objects {
		if !strings.HasPrefix(key, langPrefix) {
			continue
		}

		// "minecraft/lang/en_gb.json" -> "en-gb"
		name := strings.TrimPrefix(key, langPrefix)
		name = strings.ReplaceAll(name, "_", "-")
		name = strings.TrimSuffix(name, filepath.Ext(name))

		// For the lookup source language we need the unstripped raw map so that
		// the original key prefixes (block., item., …) are still available for
		// category classification. All other languages only need the processed form.
		if name == lookupSource {
			raw, err := fetchRawLanguage(name, obj.Hash)
			if err != nil {
				log.Error().Err(err).Str("language", name).Msg("failed to fetch lookup source language")
				return nil, fmt.Errorf("fetch lookup source %q: %w", name, err)
			}
			populateLookup(l, raw)
			saveProcessed(name, raw) // best-effort, errors are logged inside
		} else {
			if err := ensureLanguageCached(name, obj.Hash); err != nil {
				log.Error().Err(err).Str("language", name).Msg("failed to download language, skipping")
			}
		}
	}

	sort.Strings(l.Block)
	sort.Strings(l.Item)
	sort.Strings(l.Entity)
	sort.Strings(l.Custom)

	if err := l.Save(); err != nil {
		log.Warn().Err(err).Msg("failed to persist lookup cache")
	}

	return l, nil
}

// fetchAssetIndex resolves the Mojang asset manifest for the configured version.
func fetchAssetIndex() (*assetIndex, error) {
	body, err := utils.NewHttpRequest("https://piston-meta.mojang.com/mc/game/version_manifest_v2.json")
	if err != nil {
		return nil, fmt.Errorf("fetch version manifest: %w", err)
	}

	var manifest versionManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("decode version manifest: %w", err)
	}

	versionURL := ""
	for _, v := range manifest.Versions {
		if v.ID == cfg.MinecraftVersion {
			versionURL = v.URL
			break
		}
	}
	if versionURL == "" {
		return nil, fmt.Errorf("version %q not found in manifest", cfg.MinecraftVersion)
	}
	log.Info().Str("version", cfg.MinecraftVersion).Str("url", versionURL).Msg("version URL resolved")

	body, err = utils.NewHttpRequest(versionURL)
	if err != nil {
		return nil, fmt.Errorf("fetch version data: %w", err)
	}

	var ver versionData
	if err := json.Unmarshal(body, &ver); err != nil {
		return nil, fmt.Errorf("decode version data: %w", err)
	}
	if ver.AssetIndex.URL == "" {
		return nil, fmt.Errorf("asset index URL missing for version %q", cfg.MinecraftVersion)
	}
	log.Info().Str("url", ver.AssetIndex.URL).Msg("asset URL resolved")

	body, err = utils.NewHttpRequest(ver.AssetIndex.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch asset index: %w", err)
	}

	var index assetIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return nil, fmt.Errorf("decode asset index: %w", err)
	}

	return &index, nil
}

// fetchRawLanguage returns the raw (unstripped) Mojang translation map for the
// given language. It downloads from CDN if not already cached locally.
func fetchRawLanguage(name, hash string) (map[string]string, error) {
	rawPath := filepath.Join(cfg.CacheDir, "tmp", "lang-raw", cfg.MinecraftVersion,
		fmt.Sprintf("%s.json", name))

	if utils.FileExists(rawPath, true) {
		data, err := os.ReadFile(rawPath)
		if err == nil {
			var raw map[string]string
			if json.Unmarshal(data, &raw) == nil {
				log.Debug().Str("language", name).Msg("raw language loaded from tmp cache")
				return raw, nil
			}
		}
	}

	url := fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", hash[:2], hash)
	body, err := utils.NewHttpRequest(url)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}

	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}

	// Cache the raw file for future runs so we don't re-download.
	if err := utils.SaveJSONFile(rawPath, raw); err != nil {
		log.Warn().Err(err).Str("language", name).Msg("failed to cache raw language file")
	}

	return raw, nil
}

// ensureLanguageCached downloads and processes a language file if it is not
// already present in the i18n cache directory.
func ensureLanguageCached(name, hash string) error {
	outFile := filepath.Join(cfg.I18nDir(), fmt.Sprintf("%s.json", name))

	if utils.FileExists(outFile, true) {
		log.Debug().Str("language", name).Msg("language already cached")
		return nil
	}

	url := fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", hash[:2], hash)
	body, err := utils.NewHttpRequest(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}

	var raw map[string]string
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	processed := stripTranslations(raw)
	if err := utils.SaveJSONFile(outFile, processed); err != nil {
		return fmt.Errorf("save %s: %w", name, err)
	}

	log.Info().Str("language", name).Str("url", url).Msg("language cached")
	return nil
}

// saveProcessed strips and saves the processed form of an already-fetched raw map.
func saveProcessed(name string, raw map[string]string) {
	outFile := filepath.Join(cfg.I18nDir(), fmt.Sprintf("%s.json", name))

	if utils.FileExists(outFile, true) {
		return
	}

	if err := utils.SaveJSONFile(outFile, stripTranslations(raw)); err != nil {
		log.Warn().Err(err).Str("language", name).Msg("failed to save processed language file")
	}
}

// stripTranslations filters keys by processRe and cleans up values.
func stripTranslations(raw map[string]string) map[string]string {
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if !processRe.MatchString(key) {
			continue
		}
		stripped := processRe.ReplaceAllString(key, "")
		cleaned := strings.TrimSpace(strings.ReplaceAll(
			strings.ReplaceAll(val, "%s", ""), "  ", " "))
		out[stripped] = cleaned
	}
	return out
}

// populateLookup categorises raw Mojang keys into the four Lookup lists.
// It uses the original (unstripped) keys so that the block./item./... prefix
// is still present for matching.
func populateLookup(l *cache.Lookup, raw map[string]string) {
	for key := range raw {
		m := populateRe.FindStringSubmatch(key)
		if len(m) < 2 {
			continue
		}

		stripped := processRe.ReplaceAllString(key, "")

		switch m[1] {
		case "block":
			l.Block = append(l.Block, stripped)
		case "item":
			l.Item = append(l.Item, stripped)
		case "entity":
			l.Entity = append(l.Entity, stripped)
		case "stat":
			l.Custom = append(l.Custom, stripped)
		default:
			log.Warn().Str("prefix", m[1]).Str("key", key).Msg("unknown key prefix, skipping")
		}
	}
}
