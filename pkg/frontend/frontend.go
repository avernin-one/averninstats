package frontend

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
)

//go:embed files/*
var files embed.FS

func Copy() {
	err := fs.WalkDir(files, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("files", path)
		if err != nil {
			return err
		}
		dst := filepath.Join(config.Get().OutputDir, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}

		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			log.Warn().Err(err).Str("path", dst).Msg("failed to create directory")
			return nil
		}

		if err := os.WriteFile(dst, data, 0o644); err != nil {
			log.Warn().Err(err).Str("path", dst).Msg("failed to write file")
		}

		log.Debug().Str("file", dst).Msg("frontend file written")

		return nil
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to copy embedded frontend files")
	}
}
