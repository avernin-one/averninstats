package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// General
	Help     bool
	Version  bool
	Config   string
	Frontend bool

	// Logging
	LogDebug   bool `yaml:"logDebug"`
	LogJson    bool `yaml:"logJson"`
	LogNoColor bool `yaml:"logNoColor"`

	// Folders
	OutputDir      string `yaml:"outputDir"`
	CacheDir       string `yaml:"cacheDir"`
	StatsSourceDir string `yaml:"statsSourceDir"`
	Minify         bool   `yaml:"minify"`

	// Minecraft
	MinecraftVersion string `yaml:"minecraftVersion"`

	// Stats
	NumHighscores       int      `yaml:"numHighscores"`
	NumPlayerHighscores int      `yaml:"numPlayerHighscores"`
	MinPlayTime         int      `yaml:"minPlayTime"`
	CacheMaxAge         int      `yaml:"cacheMaxAge"`
	LastCheckJitter     int      `yaml:"lastCheckJitter"`
	ExcludeUUIDs        []string `yaml:"excludeUUIDs"`

	// Language
	Languages     []string `yaml:"languages"`
	ListLanguages bool     `yaml:"listLanguages"`
}

const i18nDir = "i18n"

var (
	cfg       *Config
	version   string
	gitCommit string
	buildDate string
)

// Return the config and initialize it if necessary.
func Get() *Config {
	if cfg == nil {
		Init()
	}
	return cfg
}

// Parses flags, env vars, and an optional config file. Must be called
// exactly once at program startup before any other package uses config.Get().
func Init() *Config {
	cfg = &Config{}

	cw := zerolog.ConsoleWriter{
		Out:         os.Stdout,
		TimeFormat:  time.RFC3339,
		FieldsOrder: []string{"timestamp", "level", "step", "error", "*"},
	}
	log.Logger = zerolog.New(&cw).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	// General
	pflag.BoolVar(&cfg.Help, "help", false, "Print this help message and exit")
	pflag.BoolVar(&cfg.Version, "version", false, "Print version information and exit")
	pflag.StringVar(&cfg.Config, "config", "", "Path to a optional YAML config file")
	pflag.BoolVar(&cfg.Frontend, "frontend", true, "Copy frontend files to the configured output directory")

	// Logging
	pflag.BoolVar(&cfg.LogDebug, "log-debug", false, "Enable debug logging")
	pflag.BoolVar(&cfg.LogJson, "log-json", false, "Enable JSON log format")
	pflag.BoolVar(&cfg.LogNoColor, "log-no-color", false, "Disable log colors")

	// Folders
	pflag.StringVar(&cfg.OutputDir, "output-dir", "./output", "Directory for output files")
	pflag.StringVar(&cfg.CacheDir, "cache-dir", "./cache", "Directory for cache files")
	pflag.StringVar(&cfg.StatsSourceDir, "stats-source-dir", "./stats", "Directory with per-player stats JSON files")
	pflag.BoolVar(&cfg.Minify, "minify", true, "Minify output files")

	// Minecraft
	pflag.StringVar(&cfg.MinecraftVersion, "minecraft-version", "26.1.2", "Target Minecraft version")

	// Stats
	pflag.IntVar(&cfg.NumHighscores, "num-highscores", 10, "Global highscore list size per stat")
	pflag.IntVar(&cfg.NumPlayerHighscores, "num-player-highscores", 5, "Per-player top-N scores per category")
	pflag.IntVar(&cfg.MinPlayTime, "min-play-time", 10, "Minimum playtime in minutes to include a player")
	pflag.IntVar(&cfg.CacheMaxAge, "cache-max-age", 336, "Max cache age in hours before renewal")
	pflag.IntVar(&cfg.LastCheckJitter, "last-check-jitter", 96, "Random jitter in hours added to cache expiry")
	pflag.StringSliceVar(&cfg.ExcludeUUIDs, "exclude-uuids", []string{}, "Exclude UUIDs from being processed")

	// Languages
	pflag.StringSliceVar(&cfg.Languages, "languages", []string{"en-gb"}, "Languages used in frontend for translations")
	pflag.BoolVar(&cfg.ListLanguages, "list-languages", false, "List available languages and exit")

	pflag.Parse()

	if err := env.ParseWithOptions(cfg, env.Options{
		Prefix:                "BUKI_",
		UseFieldNameByDefault: true,
	}); err != nil {
		log.Error().Err(err).Msg("failed to parse env vars")
	}

	if cfg.Help {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
		os.Exit(0)
	}

	if cfg.Version {
		fmt.Printf("Version:    %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\n",
			version, gitCommit, buildDate, runtime.Version())
		os.Exit(0)
	}

	// Apply log settings after flag/env parsing so CLI overrides env.
	if cfg.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if cfg.LogNoColor {
		cw.NoColor = true
	}
	if cfg.LogJson {
		log.Logger = zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
	}

	cfg.readFile()
	cfg.validate()

	return cfg
}

// I18nDir returns the absolute path to the i18n output directory.
func (c *Config) I18nDir() string {
	return filepath.Join(c.OutputDir, i18nDir)
}

func (c *Config) readFile() {
	if c.Config == "" {
		log.Warn().Msg("no config file specified")
		return
	}

	data, err := os.ReadFile(filepath.Clean(c.Config))
	if err != nil {
		log.Error().Err(err).Str("path", c.Config).Msg("error reading config file")
		return
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		log.Fatal().Err(err).Msg("error parsing config file")
	}
}

func (c *Config) validate() {
	dirs := []string{
		c.OutputDir,
		c.CacheDir,
		c.I18nDir(),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o750); err != nil {
			log.Fatal().Err(err).Str("path", d).Msg("failed to create directory")
		}

		log.Debug().Str("path", d).Msg("directory ready")
	}
}
