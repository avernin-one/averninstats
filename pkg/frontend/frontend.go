package frontend

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/avernin-one/averninstats/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
)

//go:embed all:files/*
var files embed.FS

var m *minify.M

type Frontend struct{}

func New() *Frontend {
	if config.Get().Minify {
		m = minify.New()

		tmplMinifier := &html.Minifier{}
		tmplMinifier.TemplateDelims = [2]string{"{{", "}}"}

		m.Add("mustache", tmplMinifier)
		m.AddFunc("css", css.Minify)
		m.AddFunc("html", html.Minify)
		m.AddFunc("js", js.Minify)
	}

	return &Frontend{}
}

func (f *Frontend) Copy() {
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
			return os.MkdirAll(dst, 0o750)
		}

		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}

		if config.Get().Minify {
			data = minifyFile(path, data)
		}

		if err := os.WriteFile(dst, data, 0o600); err != nil {
			log.Warn().Err(err).Str("path", dst).Msg("failed to write file")
			return nil
		}

		log.Debug().Str("file", dst).Msg("frontend file written")
		return nil
	})

	if err != nil {
		log.Fatal().Err(err).Msg("failed to copy embedded frontend files")
	}
}

func minifyFile(path string, data []byte) []byte {
	mediatype := mediaType(filepath.Ext(path))
	if mediatype == "" {
		return data
	}

	minified, err := m.Bytes(mediatype, data)
	if err != nil {
		log.Warn().Err(err).Str("path", path).Msg("minification failed, using original")
		return data
	}

	return minified
}

func mediaType(ext string) string {
	switch ext {
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".js":
		return "js"
	case ".mustache":
		return "mustache"
	default:
		return ""
	}
}
