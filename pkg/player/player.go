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

	"github.com/avernin-one/averninstats/pkg/utils"
	skin "github.com/mineatar-io/skin-render"
	"github.com/rs/zerolog/log"
)

// Holds the resolved profile data for a single Minecraft player.
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

type Metadata struct {
	Textures struct {
		Skin struct {
			URL      string `json:"url"`
			Metadata struct {
				Model string `json:"model"`
			} `json:"metadata"`
		} `json:"SKIN"` //nolint:tagliatelle // defined by mojoang
	} `json:"textures"`
}

// Retrieves profile and skin metadata for uuid from the Mojang API.
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

	var meta Metadata
	if encodedTexture != "" {
		decoded, err := base64.StdEncoding.DecodeString(encodedTexture)
		if err != nil {
			log.Warn().Str("uuid", uuid).Err(err).Msg("unable to decode base64 texture")
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

	return d, nil
}

// GetSkin downloads the skin image from url. Falls back to the default skin
// on failure.
func GetSkin(url string) image.Image {
	img, err := downloadImage(url)
	if err != nil {
		log.Warn().Str("url", url).Err(err).Msg("unable to download skin, using default")
		return skin.GetDefaultSkin(true)
	}

	return img
}

// SaveHead renders and saves the face/head image for the given skin.
func SaveHead(img image.Image, outputDir, playerName, playerModel string) {
	path := headPath(outputDir, playerName)

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		log.Warn().Str("player", playerName).Msg("unexpected image type for head render")
		return
	}

	rendered := skin.RenderFace(nrgba, skin.Options{
		Scale:   1,
		Overlay: true,
		Slim:    playerModel == "slim",
		Square:  true,
	})
	if err := saveImage(rendered, path); err != nil {
		log.Warn().Str("player", playerName).Err(err).Msg("unable to save head image")
		return
	}

	log.Debug().Str("player", playerName).Msg("saved player head")
}

// Renders and saves the full-body image for the given skin.
func SaveBody(img image.Image, outputDir, playerName, playerModel string) {
	path := bodyPath(outputDir, playerName)

	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		log.Warn().Str("player", playerName).Msg("unexpected image type for body render")
		return
	}

	rendered := skin.RenderBody(nrgba, skin.Options{
		Scale:   7,
		Overlay: true,
		Slim:    playerModel == "slim",
		Square:  false,
	})
	if err := saveImage(rendered, path); err != nil {
		log.Warn().Str("player", playerName).Err(err).Msg("unable to save body image")
		return
	}

	log.Debug().Str("player", playerName).Msg("saved player body")
}

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

func saveImage(img image.Image, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	return nil
}

func HeadExists(outputDir, playerName string) bool {
	return utils.FileExists(headPath(outputDir, playerName), true)
}

func BodyExists(outputDir, playerName string) bool {
	return utils.FileExists(bodyPath(outputDir, playerName), true)
}

func headPath(outputDir, playerName string) string {
	return filepath.Join(outputDir, "assets", "images", "player", fmt.Sprintf("head_%s.png", playerName))
}

func bodyPath(outputDir, playerName string) string {
	return filepath.Join(outputDir, "assets", "images", "player", fmt.Sprintf("body_%s.png", playerName))
}
