# syntax=docker/dockerfile:1
FROM gcr.io/distroless/static-debian12:nonroot

ARG APP_NAME=averninstats

COPY dist/${APP_NAME} /usr/local/bin/${APP_NAME}

ENTRYPOINT ["/usr/local/bin/averninstats"]
