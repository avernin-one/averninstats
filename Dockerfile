# syntax=docker/dockerfile:1
FROM debian:13-slim

ARG APP_NAME=averninstats

COPY dist/${APP_NAME} /usr/local/bin/${APP_NAME}

RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot \
    && chown -R nonroot:nonroot /usr/local/bin \
    && chmod -R +x /usr/local/bin

USER nonroot

ENTRYPOINT ["/usr/local/bin/averninstats"]
