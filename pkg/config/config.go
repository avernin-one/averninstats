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
	Help    bool
	Version bool
	Config  string

	// LOGGING
	LogDebug   bool `yaml:"logDebug"`
	LogJson    bool `yaml:"logJson"`
	LogNoColor bool `yaml:"logNoColor"`

	// FOLDERS
	OutputDir      string `yaml:"outputDir"`
	CacheDir       string `yaml:"cacheDir"`
	StatsSourceDir string `yaml:"statsSourceDir"`

	// MINECRAFT
	MinecraftVersion string

	// STATS
	NumHighscores       int  `yaml:"numHighscores"`
	NumPlayerHighscores int  `yaml:"numPlayerHighscores"`
	MinPlayTime         int  `yaml:"minPlayTime"`
	CacheMaxAge         int  `yaml:"cacheMaxAge"`
	LastCheckJitter     int  `yaml:"lastCheckJitter"`
	QueryDelay          int  `yaml:"queryDelay"`
	NoDelete            bool `yaml:"noDelete"`
}

const (
	i18nDir = "i18n"
)

var (
	cfg *Config

	version   string
	gitCommit string
	buildDate string
)

func Get() *Config {
	if cfg == nil {
		Init()
	}

	return cfg
}

func Init() {
	cfg = &Config{}

	//	LOGGER
	cw := zerolog.ConsoleWriter{
		Out:         os.Stdout,
		TimeFormat:  time.RFC3339,
		FieldsOrder: []string{"timestamp", "level", "step", "error", "*"},
	}

	log.Logger = zerolog.New(&cw).With().Caller().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	//	PFLAGS
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	pflag.BoolVar(&cfg.Help, "help", false, "Print this help message and exit")
	pflag.BoolVar(&cfg.Version, "version", false, "Print version information and exit")
	pflag.StringVar(&cfg.Config, "config", "", "Path to the configuration YAML file")

	// FOLDERS
	pflag.StringVar(&cfg.OutputDir, "output-dir", "./output", "Directory where output files will be saved")
	pflag.StringVar(&cfg.CacheDir, "cache-dir", "./cache", "Directory where cache files will be saved")
	pflag.StringVar(&cfg.StatsSourceDir, "stats-source-dir", "./stats", "Directory containing Minecraft per-player stats JSON files")

	// LOGGING
	pflag.BoolVar(&cfg.LogDebug, "log-debug", false, "Enable verbose debug logging")
	pflag.BoolVar(&cfg.LogJson, "log-json", false, "Enable json logging output formt")
	pflag.BoolVar(&cfg.LogNoColor, "log-no-color", false, "Disable text colors for logging")

	// MINECRAFT
	pflag.StringVar(&cfg.MinecraftVersion, "minecraft-version", "1.21.11", "Minecraft version to target for stats processing")

	// STATS
	pflag.IntVar(&cfg.NumHighscores, "num-highscores", 10, "Number of top global highscores to track for each statistic")
	pflag.IntVar(&cfg.NumPlayerHighscores, "num-player-highscores", 5, "Number of top personal highscores to track for each player and statistic")
	pflag.IntVar(&cfg.MinPlayTime, "min-play-time", 10, "Minimum play time in minutes for a player to be included in the stats")
	pflag.IntVar(&cfg.CacheMaxAge, "cache-max-age", 336, "Maximum age in hours for cached player data before it's renewed")
	pflag.IntVar(&cfg.LastCheckJitter, "last-check-jitter", 96, "Maximum random jitter in hours to add to player last check time to avoid cache stampedes")
	pflag.IntVar(&cfg.QueryDelay, "query-delay", 2, "Number of seconds to wait between Mojang API queries to avoid rate limits (should not be <2)")
	pflag.BoolVar(&cfg.NoDelete, "no-delete", false, "Don't delete any existing output files, only add new ones or update existing ones")

	pflag.Parse()

	// PARSE ENV VARS
	options := env.Options{
		Prefix:                "BUKI_",
		UseFieldNameByDefault: true,
	}

	err := env.ParseWithOptions(cfg, options)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse env vars")
	}

	if cfg.Help {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
		os.Exit(2)
	}

	if cfg.Version {
		fmt.Printf("Version:    %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\n",
			version,
			gitCommit,
			buildDate,
			runtime.Version())
		os.Exit(0)
	}

	if cfg.LogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if cfg.LogNoColor {
		cw.NoColor = cfg.LogNoColor
	}

	if cfg.LogJson {
		log.Logger = zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
	}

	read()
	validate()
}

func validate() {
	if err := os.MkdirAll(filepath.Join(cfg.OutputDir), 0o775); err != nil {
		log.Fatal().Err(err).Str("output_dir", cfg.OutputDir).Msg("failed to create output directory")
	} else {
		log.Info().Str("output_dir", cfg.OutputDir).Msg("output directory present")
	}

	if err := os.MkdirAll(filepath.Join(cfg.CacheDir), 0o775); err != nil {
		log.Fatal().Err(err).Str("cache_dir", cfg.CacheDir).Msg("failed to create cache directory")
	} else {
		log.Info().Str("cache_dir", cfg.CacheDir).Msg("cache directory present")
	}

	if err := os.MkdirAll(filepath.Join(cfg.OutputDir, i18nDir), 0o775); err != nil {
		log.Fatal().Err(err).Msg("failed to create i18n output directory")
	} else {
		log.Info().Str("i18n_dir", filepath.Join(cfg.OutputDir, i18nDir)).Msg("i18n output directory present")
	}
}

func read() {
	if cfg.Config == "" {
		log.Info().Msg("no config file specified")
		return
	}

	data, err := os.ReadFile(filepath.Clean(cfg.Config))
	if err != nil {
		log.Error().Err(err).Msg("error reading config file")
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error unmarshalling config file")
	}
}

func (c *Config) I18nDir() string {
	return filepath.Join(c.OutputDir, i18nDir)
}
