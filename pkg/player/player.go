package player

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/avernin-one/averninstats/pkg/utils"
	skin "github.com/mineatar-io/skin-render"
	"github.com/rs/zerolog/log"
)

// Data holds the raw profile response from the Mojang session server.
type Data struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"properties"`
	SkinURL   string `json:"-"`
	SkinModel string `json:"-"`
}

// playerMetadata is the base64-decoded texture payload inside Properties.
type playerMetadata struct {
	Textures struct {
		Skin struct {
			URL      string `json:"url"`
			Metadata struct {
				Model string `json:"model"`
			} `json:"metadata"`
		} `json:"SKIN"`
	} `json:"textures"`
}

var cfg = config.Get()

// Fetch retrieves profile and skin metadata for the given UUID from Mojang,
// waits cfg.QueryDelay seconds, and returns the resolved Data.
func Fetch(uuid string) (Data, error) {
	url := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/profile/%s", uuid)

	body, err := utils.NewHttpRequest(url)
	if err != nil {
		return Data{}, fmt.Errorf("fetch profile %q: %w", uuid, err)
	}

	var d Data
	if err := json.Unmarshal(body, &d); err != nil {
		return Data{}, fmt.Errorf("decode profile %q: %w", uuid, err)
	}

	var encodedTexture string
	for _, prop := range d.Properties {
		if prop.Name == "textures" {
			encodedTexture = prop.Value
			break
		}
	}

	var meta playerMetadata
	if encodedTexture != "" {
		decoded, err := base64.StdEncoding.DecodeString(encodedTexture)
		if err != nil {
			log.Warn().Str("uuid", uuid).Err(err).Msg("unable to decode base64 texture string")
		} else if err := json.Unmarshal(decoded, &meta); err != nil {
			log.Warn().Str("uuid", uuid).Err(err).Msg("unable to decode texture metadata")
		}
	}

	if meta.Textures.Skin.Metadata.Model == "" {
		meta.Textures.Skin.Metadata.Model = "wide"
	}

	d.SkinURL = meta.Textures.Skin.URL
	d.SkinModel = meta.Textures.Skin.Metadata.Model

	log.Info().Str("uuid", uuid).Str("name", d.Name).Msg("resolved UUID")
	time.Sleep(time.Duration(cfg.QueryDelay) * time.Second)

	return d, nil
}

// GetSkin downloads the skin image from url. Falls back to the default skin
// if the download fails.
func GetSkin(url string) image.Image {
	img, err := downloadImage(url)
	if err != nil {
		log.Warn().Str("url", url).Err(err).Msg("unable to download player skin, using default")
		return skin.GetDefaultSkin(true)
	}
	return img
}

// SaveHead renders and saves the face/head image for the given skin.
func SaveHead(img image.Image, outputDir, playerName, playerModel string) {
	filePath := filepath.Join(outputDir, "assets", "images", "players",
		fmt.Sprintf("head_%s.png", playerName))

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		log.Warn().Str("player", playerName).Msg("unexpected image type for player head")
		return
	}

	rendered := skin.RenderFace(nrgba, skin.Options{
		Scale:   12,
		Overlay: true,
		Slim:    playerModel == "slim",
		Square:  true,
	})

	if err := saveImage(rendered, filePath); err != nil {
		log.Warn().Str("player", playerName).Err(err).Msg("unable to save player head image")
		return
	}
	log.Debug().Str("player", playerName).Str("model", playerModel).Msg("saved player head")
}

// SaveBody renders and saves the full-body image for the given skin.
func SaveBody(img image.Image, outputDir, playerName, playerModel string) {
	filePath := filepath.Join(outputDir, "assets", "images", "players",
		fmt.Sprintf("body_%s.png", playerName))

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		log.Warn().Str("player", playerName).Msg("unexpected image type for player body")
		return
	}

	rendered := skin.RenderBody(nrgba, skin.Options{
		Scale:   10,
		Overlay: true,
		Slim:    playerModel == "slim",
		Square:  false,
	})

	if err := saveImage(rendered, filePath); err != nil {
		log.Warn().Str("player", playerName).Err(err).Msg("unable to save player body image")
		return
	}
	log.Debug().Str("player", playerName).Str("model", playerModel).Msg("saved player body")
}

// HeadExists reports whether the rendered head image for playerName exists.
func HeadExists(outputDir, playerName string) bool {
	return utils.FileExists(filepath.Join(outputDir, "assets", "images", "players",
		fmt.Sprintf("head_%s.png", playerName)), true)
}

// BodyExists reports whether the rendered body image for playerName exists.
func BodyExists(outputDir, playerName string) bool {
	return utils.FileExists(filepath.Join(outputDir, "assets", "images", "players",
		fmt.Sprintf("body_%s.png", playerName)), true)
}

// downloadImage fetches and decodes an image from url.
func downloadImage(url string) (image.Image, error) {
	data, err := utils.NewHttpRequest(url)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image from %q: %w", url, err)
	}
	return img, nil
}

// saveImage encodes img as PNG and writes it to filePath.
func saveImage(img image.Image, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o775); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o664)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}
	return nil
}
