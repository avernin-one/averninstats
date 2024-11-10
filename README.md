# averninstats

## Demo

You can view a live demo (hosted on github.com) at the following URL:

[https://stats.avernin.one](https://stats.avernin.one)


## Prerequisites

- [Git](https://git-scm.com/downloads)
- [Docker](https://docs.docker.com/manuals)

## HowTo

```
$ git clone https://github.com/avernin-one/averninstats.git
$ cd averninstats
$ docker run \
    --rm \
    --user $(id -u):$(id -g) \
    -v "/path/to/world/stats/:/source:ro" \
    -v ".:/output:rw" \
    linogics/averninstats:latest \
    -url stats.avernin.one \
$ docker run \
    --rm \
    --user $(id -u):$(id -g) \
    -v ".:/src" \
    hugomods/hugo:exts-non-root \
    --minify

# ALTERNATIVE: if you want to access the generated website locally
$ docker run \
    --rm \
    --user $(id -u):$(id -g) \
    -v ".:/src" \
    -p 1313:1313 \
    hugomods/hugo:exts-non-root \
    server
```
