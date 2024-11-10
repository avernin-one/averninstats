# averninstats

## Demo


## Steps
```
    $> git clone https://github.com/avernin-one/averninstats.git
    $> cd averninstats
    $> docker run -it --rm --user $(id -u):$(id -g) \
        -v "/path/to/server/world/stats:/source:ro" \
        -v ".:/output:rw" \
        linogics/averninstats:latest \
        -url stats.avernin.one

    $> hugo --minify
```
