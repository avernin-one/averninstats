# averninstats

## Demo

You can view a live demo (hosted on github.com) at the following URL:

[https://stats.avernin.one](https://stats.avernin.one)


## Prerequisites

- [Git](https://git-scm.com/downloads)
- [Hugo](https://gohugo.io/installation) (extended)
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
$ hugo --minify

# or if you want to access the rendered pages locally
$ hugo server
```
