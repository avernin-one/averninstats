# syntax=docker/dockerfile:1
FROM debian:13-slim

ARG APP_NAME=averninstats

COPY dist/${APP_NAME} /usr/local/bin/${APP_NAME}

ENTRYPOINT ["/usr/local/bin/averninstats"]
