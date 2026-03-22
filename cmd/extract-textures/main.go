// extract-textures extracts block, item and entity preview images from the
// Minecraft client JAR. The JAR is downloaded automatically from Mojang's
// official API if it is not already present in the output directory.
//
// Usage:
//
//	go run ./cmd/extract-textures [flags]
//
// Flags:
//
//	-version   Minecraft version to use (default "1.21.1")
//	-out       Output directory for extracted images (default "./output")
//	-jar       Path to an already-downloaded client JAR (skips download)
//	-force     Re-extract even if images already exist
package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const versionManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

var (
	flagVersion = flag.String("version", "1.21.1", "Minecraft version to extract textures for")
	flagOut     = flag.String("out", "./output", "Output directory for extracted images")
	flagJar     = flag.String("jar", "", "Path to an existing client JAR (skips download)")
	flagForce   = flag.Bool("force", false, "Re-extract even if images already exist")
)

// ---------------------------------------------------------------------------
// Mojang API types
// ---------------------------------------------------------------------------

type versionManifest struct {
	Versions []versionEntry `json:"versions"`
}

type versionEntry struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type versionMeta struct {
	Downloads struct {
		Client struct {
			URL  string `json:"url"`
			SHA1 string `json:"sha1"`
			Size int64  `json:"size"`
		} `json:"client"`
	} `json:"downloads"`
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func main() {
	flag.Parse()
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run() error {
	jarPath := *flagJar

	if jarPath == "" {
		var err error
		jarPath, err = ensureJar(*flagVersion, *flagOut)
		if err != nil {
			return err
		}
	}

	outDir := filepath.Join(*flagOut, "assets", "images")

	dirs := []string{
		filepath.Join(outDir, "blocks"),
		filepath.Join(outDir, "items"),
		filepath.Join(outDir, "entities"),
	}
	for _, d := range dirs {
		if !*flagForce && dirHasFiles(d) {
			log.Printf("skipping extraction, %s already has files (use -force to re-run)", d)
			return nil
		}
		if err := os.MkdirAll(d, 0o775); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}

	log.Printf("opening JAR: %s", jarPath)
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return fmt.Errorf("open jar: %w", err)
	}
	defer r.Close()

	// Build a path -> entry map for fast lookup.
	byPath := make(map[string]*zip.File, len(r.File))
	for _, f := range r.File {
		byPath[f.Name] = f
	}

	// Extract all block, item and entity textures we can find.
	if err := extractAll(byPath, outDir); err != nil {
		return err
	}

	log.Printf("done. images written to %s", outDir)
	return nil
}

// ---------------------------------------------------------------------------
// JAR download
// ---------------------------------------------------------------------------

// ensureJar downloads the client JAR for the given version if it is not
// already present in <outDir>/jars/<version>.jar and returns its path.
func ensureJar(version, outDir string) (string, error) {
	jarDir := filepath.Join(outDir, "jars")
	if err := os.MkdirAll(jarDir, 0o775); err != nil {
		return "", fmt.Errorf("create jar dir: %w", err)
	}

	jarPath := filepath.Join(jarDir, version+".jar")
	if _, err := os.Stat(jarPath); err == nil {
		log.Printf("JAR already present: %s", jarPath)
		return jarPath, nil
	}

	log.Printf("fetching version manifest...")
	meta, err := fetchVersionMeta(version)
	if err != nil {
		return "", err
	}

	log.Printf("downloading Minecraft %s client JAR (%s)...", version, formatBytes(meta.Downloads.Client.Size))
	if err := downloadFile(meta.Downloads.Client.URL, jarPath, meta.Downloads.Client.SHA1); err != nil {
		return "", fmt.Errorf("download jar: %w", err)
	}

	return jarPath, nil
}

// fetchVersionMeta fetches the version-specific meta JSON from Mojang's API.
func fetchVersionMeta(version string) (*versionMeta, error) {
	manifest, err := httpGetJSON[versionManifest](versionManifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch version manifest: %w", err)
	}

	var metaURL string
	for _, v := range manifest.Versions {
		if v.ID == version {
			metaURL = v.URL
			break
		}
	}
	if metaURL == "" {
		return nil, fmt.Errorf("version %q not found in manifest", version)
	}

	meta, err := httpGetJSON[versionMeta](metaURL)
	if err != nil {
		return nil, fmt.Errorf("fetch version meta: %w", err)
	}
	return meta, nil
}

// downloadFile downloads url to dst and verifies the SHA1 checksum.
func downloadFile(url, dst, expectedSHA1 string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	h := sha1.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	got := hex.EncodeToString(h.Sum(nil))
	if got != expectedSHA1 {
		os.Remove(tmp)
		return fmt.Errorf("SHA1 mismatch: got %s, want %s", got, expectedSHA1)
	}

	return os.Rename(tmp, dst)
}

// httpGetJSON fetches a URL and decodes the JSON body into T.
func httpGetJSON[T any](url string) (*T, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Texture extraction
// ---------------------------------------------------------------------------

// extractAll walks the ZIP and extracts every block, item and entity texture.
func extractAll(byPath map[string]*zip.File, outDir string) error {
	blockPrefix := "assets/minecraft/textures/block/"
	itemPrefix := "assets/minecraft/textures/item/"
	entityPrefix := "assets/minecraft/textures/entity/"

	for path, zf := range byPath {
		if !strings.HasSuffix(path, ".png") {
			continue
		}

		switch {
		case strings.HasPrefix(path, blockPrefix):
			name := strings.TrimSuffix(strings.TrimPrefix(path, blockPrefix), ".png")
			// Skip sub-directory entries (some packs have folders inside block/).
			if strings.Contains(name, "/") {
				continue
			}
			dst := filepath.Join(outDir, "blocks", name+".png")
			if err := extractBlockOrItem(zf, dst); err != nil {
				log.Printf("warn: block %s: %v", name, err)
			}

		case strings.HasPrefix(path, itemPrefix):
			name := strings.TrimSuffix(strings.TrimPrefix(path, itemPrefix), ".png")
			if strings.Contains(name, "/") {
				continue
			}
			dst := filepath.Join(outDir, "items", name+".png")
			if err := extractBlockOrItem(zf, dst); err != nil {
				log.Printf("warn: item %s: %v", name, err)
			}

		case strings.HasPrefix(path, entityPrefix):
			// Derive a clean entity name from the path.
			// e.g. "assets/minecraft/textures/entity/creeper/creeper.png" -> "creeper"
			rel := strings.TrimPrefix(path, entityPrefix)
			name := entityName(rel)
			if name == "" {
				continue
			}
			dst := filepath.Join(outDir, "entities", name+".png")
			// Only write the first match per entity name.
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			if err := extractEntityPreview(zf, dst); err != nil {
				log.Printf("warn: entity %s: %v", name, err)
			}
		}
	}

	return nil
}

// extractBlockOrItem copies a block or item PNG, cropping to the first frame
// if the texture is animated (height > width).
func extractBlockOrItem(zf *zip.File, dst string) error {
	img, err := readZipImage(zf)
	if err != nil {
		return err
	}

	// Animated textures store frames vertically. Keep only the first frame.
	img = cropToSquare(img)

	return writePNG(dst, img)
}

// extractEntityPreview crops a face/head preview from an entity skin sheet.
func extractEntityPreview(zf *zip.File, dst string) error {
	img, err := readZipImage(zf)
	if err != nil {
		return err
	}

	preview := cropEntityFace(img)
	return writePNG(dst, preview)
}

// entityName derives a clean name from a relative entity texture path.
// "creeper/creeper.png"  -> "creeper"
// "cow/mooshroom.png"    -> "cow_mooshroom"
// "zombie.png"           -> "zombie"
func entityName(rel string) string {
	rel = strings.TrimSuffix(rel, ".png")
	parts := strings.Split(rel, "/")

	switch len(parts) {
	case 1:
		return parts[0]
	case 2:
		// If folder and file share the same name, use just the name.
		if parts[0] == parts[1] {
			return parts[0]
		}
		return parts[0] + "_" + parts[1]
	default:
		// Deeper nesting — skip, these are usually overlays or variants.
		return ""
	}
}

// ---------------------------------------------------------------------------
// Image helpers
// ---------------------------------------------------------------------------

// cropEntityFace extracts the face region from a standard Minecraft skin sheet.
// On a 64x64 sheet the face front is at UV (8,8)-(16,16).
// We scale the coordinates relative to the actual image width.
func cropEntityFace(src image.Image) image.Image {
	b := src.Bounds()
	w := b.Max.X

	if w >= 16 {
		unit := w / 8
		return cropImage(src, image.Rect(unit, unit, unit*2, unit*2))
	}

	return cropToSquare(src)
}

// cropToSquare returns the top-left square of the image (width x width).
// This removes extra frames from animated textures.
func cropToSquare(src image.Image) image.Image {
	b := src.Bounds()
	size := b.Max.X
	if b.Max.Y < size {
		size = b.Max.Y
	}
	if size == b.Max.Y {
		return src // already square
	}
	return cropImage(src, image.Rect(0, 0, size, size))
}

// cropImage copies the given region of src into a new RGBA image.
func cropImage(src image.Image, crop image.Rectangle) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(dst, dst.Bounds(), src, crop.Min, draw.Src)
	return dst
}

// readZipImage decodes a PNG from a ZIP entry.
func readZipImage(zf *zip.File) (image.Image, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, fmt.Errorf("open zip entry: %w", err)
	}
	defer rc.Close()

	img, _, err := image.Decode(rc)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// writePNG encodes img as PNG and writes it to dst.
func writePNG(dst string, img image.Image) error {
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ---------------------------------------------------------------------------
// Misc helpers
// ---------------------------------------------------------------------------

// dirHasFiles returns true if dir exists and contains at least one file.
func dirHasFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}

// formatBytes returns a human-readable file size string.
func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
