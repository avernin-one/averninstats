# syntax=docker/dockerfile:1
ARG OS
ARG ARCH

FROM --platform=$OS/$ARCH debian

COPY /dist/averninstats-$OS-$ARCH /app/averninstats

ENTRYPOINT /app/averninstats
