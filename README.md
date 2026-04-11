# averninstats

![Build Status](https://github.com/avernin-one/averninstats/actions/workflows/release.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/avernin-one/averninstats)](https://goreportcard.com/report/github.com/avernin-one/averninstats)
![Go version](https://img.shields.io/github/go-mod/go-version/avernin-one/averninstats)
[![Go Reference](https://pkg.go.dev/badge/github.com/avernin-one/averninstats.svg)](https://pkg.go.dev/github.com/avernin-one/averninstats)

A stats processor for Minecraft Java Edition servers. It reads the per-player
stats JSON files that Minecraft writes to disk, resolves player names and skins
via the Mojang API, and generates a static website with highscores, per-block,
per-item, per-entity breakdowns and individual player profiles.

## Preview

<table>
  <tr>
    <th align="left">Highscore</th>
    <th align="left">Player</th>
  </tr>
  <tr>
    <td align="center">
      <a href="https://github.com/user-attachments/assets/7e621ccf-720e-48bc-a97d-ebe314fe86b6">
        <img src="https://github.com/user-attachments/assets/7e621ccf-720e-48bc-a97d-ebe314fe86b6" />
      </a>
    </td>
    <td align="center">
      <a href="https://github.com/user-attachments/assets/eccaca99-32f9-458c-a516-11112fc4b064">
        <img src="https://github.com/user-attachments/assets/eccaca99-32f9-458c-a516-11112fc4b064" />
      </a>
    </td>
  </tr>
</table>

## Command-Line Options

```text
$ ./averninstats --help
Usage of ./averninstats:
      --cache-dir string            Directory for cache files (default "./cache")
      --cache-max-age int           Max cache age in hours before renewal (default 336)
      --config string               Path to a optional YAML config file
      --help                        Print this help message and exit
      --languages strings           Languages used in frontend for translations (default [en-gb])
      --last-check-jitter int       Random jitter in hours added to cache expiry (default 96)
      --list-languages              List available languages and exit
      --log-debug                   Enable debug logging
      --log-json                    Enable JSON log format
      --log-no-color                Disable log colors
      --min-play-time int           Minimum playtime in minutes to include a player (default 10)
      --minecraft-version string    Target Minecraft version (default "1.21.11")
      --minify                      Minify output files (default true)
      --no-delete                   Keep existing output files instead of clearing them
      --num-highscores int          Global highscore list size per stat (default 10)
      --num-player-highscores int   Per-player top-N scores per category (default 5)
      --output-dir string           Directory for output files (default "./output")
      --stats-source-dir string     Directory with per-player stats JSON files (default "./stats")
      --version                     Print version information and exit
```

## How to use

### Docker (recommended)

```bash
docker pull ghcr.io/avernin-one/averninstats
```

```bash
docker run --rm \
  -v /path/to/minecraft/world/stats:/stats:ro \
  -v /path/to/output:/output \
  -v /path/to/cache:/cache \
  ghcr.io/avernin-one/averninstats \
  --stats-source-dir /stats \
  --output-dir /output \
  --cache-dir /cache \
  --minecraft-version 1.21.1
```

The output directory will contain a ready-to-serve static website. Point any
web server (nginx, caddy, apache) at it, or push it into a github repository
and serve it via github-pages.

### Download Release

```bash
@TODO
```

### From source

```bash
git clone https://github.com/avernin-one/averninstats
cd averninstats
go build -o averninstats .
```

```bash
./averninstats \
  --stats-source-dir /path/to/world/stats \
  --output-dir ./output \
  --cache-dir ./cache \
  --minecraft-version 1.21.1
```
