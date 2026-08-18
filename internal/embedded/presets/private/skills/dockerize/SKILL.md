---
name: dockerize
description: Containerizing any project - multi-stage builds, choosing a base image, packaging the runtime, clearing package-manager caches, and running as a non-root user. Use when writing or changing a Dockerfile, picking a base image, deciding between alpine and debian-slim, adding a docker-compose file, or when a container runs as root. Triggers on Dockerfile, docker-compose.yaml, FROM, multi-stage builds, apk add, apt-get install, USER, adduser, groupadd, ENTRYPOINT, EXPOSE, musl, and glibc.
user-invocable: false
---

# Dockerize

**Two stages, a runtime packaged rather than assumed where the language allows it, caches cleared inside the layer that made them, and a fixed non-root user.**

This applies to any language. What changes between them is the builder image and how the runtime gets into the final stage.

## Rules

A build has exactly two stages. The builder carries the compilers, package managers, and source; the final stage carries the artifact and its runtime dependencies and nothing else. Copying the builder wholesale ships a toolchain, the full source, and every intermediate object to production, which is both a large image and a large attack surface.

A package manager's cache is cleared in the same `RUN` that created it. A separate `RUN rm -rf` writes a new layer that deletes files the previous layer still contains, so the image keeps both and grows rather than shrinks.

```dockerfile
# alpine
RUN apk add --no-cache ca-certificates tzdata

# debian
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
```

Dependency manifests are copied and installed before the source is copied, so an edit to a source file does not invalidate the dependency layer and every rebuild reinstalls nothing.

The final stage runs as a fixed, hardcoded user and group. Root inside a container is root on the host kernel, and a mounted volume written as root leaves files the host user cannot delete.

The UID and GID are pinned to a literal number rather than left to the distribution's next-available. A number that shifts between base image versions changes the ownership of every mounted volume on the next rebuild.

`EXPOSE` documents the port. It publishes nothing on its own; the actual mapping happens at run time.

`ENTRYPOINT` names the executable, which does not change, and `CMD` supplies the default arguments, which a `docker run` can override.

## Packaging the Runtime

Where a language can produce a self-contained artifact, produce one and put it on a plain base image. The image then has one thing to keep patched instead of two, and the runtime version is a property of the build rather than of whichever tag the registry resolves today.

| Language | Self-contained artifact | Final base |
|---|---|---|
| Go | `CGO_ENABLED=0` static binary | `alpine` |
| Rust | static binary, or a glibc binary | `alpine` or `debian:*-slim` |
| Node, pure JS | the runtime vendored into a directory beside the app and executed directly | `debian:*-slim` |
| Node, native addon | none, the addon is glibc-linked | `debian:*-slim` with the runtime vendored |
| Python | none in the general case | `python:*-slim` |
| Java | a jlink image, or a fat jar | `eclipse-temurin:*-jre-alpine`, or `*-jre` |

When a self-contained artifact is not possible, use the slim or alpine variant of that language's official runtime image, matching the libc the artifact was built against. A full runtime image carries build tooling the running process never uses.

Vendoring a runtime means copying the interpreter into the image and invoking it by path, rather than depending on one being installed and on `PATH`:

```dockerfile
COPY --from=builder /app/runtime /app/runtime
ENTRYPOINT ["/app/runtime/bin/node", "/app/bin/app.js"]
```

## alpine or debian-slim

The deciding question is whether anything in the image is linked against glibc.

A Go binary built with `CGO_ENABLED=0` links against nothing at all. It contains its own runtime, makes syscalls directly, and runs identically on musl and glibc, so alpine is the correct base and gives the smallest image.

A compiled native addon is the opposite case. A Node N-API addon, or a Python wheel with a compiled extension, is built against glibc, and alpine's musl cannot load it. The failure appears at run time as a missing symbol rather than at build time, which is why the choice has to be made deliberately rather than discovered.

The official Node binary is itself glibc-linked, so vendoring it into an alpine image fails for the same reason even when the application is pure JavaScript.

Alpine remains viable for those cases only with a deliberate musl matrix: a musl build of the runtime plus every native dependency recompiled against musl inside an alpine builder. That is a real option, not a forbidden one, but it is a decision to take on purpose. Copying a glibc artifact into an alpine image is never one.

## Go Template

```dockerfile
FROM golang:1.26.6-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git curl make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# make assets populates the tree that //go:embed static needs; drop it when the
# project has no embedded frontend.
RUN make assets && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/[APP_NAME] .

FROM alpine:3.24.1

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 10001 -S app && \
    adduser -u 10001 -S -G app app

WORKDIR /app
COPY --from=builder --chown=10001:10001 /app/[APP_NAME] .

RUN mkdir -p /data && chown 10001:10001 /data
VOLUME ["/data"]

USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["./[APP_NAME]"]
CMD ["serve", "-d", "/data", "-H", "0.0.0.0"]
```

`ca-certificates` is what lets the binary make an outbound HTTPS request; without it every TLS handshake fails verification. `tzdata` is what `time.LoadLocation` reads.

`--chown` on the `COPY` sets ownership as the layer is written, rather than adding a second layer that duplicates every copied file with new metadata.

`USER` is set after the last `RUN` that needs to write, since a `RUN` after it executes unprivileged and cannot create the directories the app needs.

## Node Template

```dockerfile
FROM debian:trixie-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
      curl ca-certificates build-essential python3 unzip xz-utils make \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Vendor the runtime so the final stage depends on no installed node.
ARG NODE_VERSION=24.19.0
RUN curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" \
      -o /tmp/node.tar.xz \
    && mkdir -p /app/runtime \
    && tar -xJf /tmp/node.tar.xz -C /app/runtime --strip-components=1 \
    && rm /tmp/node.tar.xz
ENV PATH="/app/runtime/bin:${PATH}"

COPY package.json package-lock.json ./
RUN npm_config_build_from_source=true npm ci --omit=dev

COPY . .
RUN make vendor

FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 10001 app \
    && useradd -u 10001 -g app -M -s /usr/sbin/nologin app

WORKDIR /app
COPY --from=builder --chown=10001:10001 /app /app

RUN mkdir -p /data && chown 10001:10001 /data
VOLUME ["/data"]

USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/runtime/bin/node", "/app/bin/[APP_NAME].js"]
CMD ["--config", "/data/config.json"]
```

The runtime is downloaded in the builder and copied forward, so the final image needs no Node installation and the version is fixed by the build argument rather than by a base image tag.

`npm ci --omit=dev` installs only what the process runs. Development dependencies in a production image are packages to patch that nothing imports.

`build-essential` and `python3` are in the builder for `node-gyp` and appear nowhere in the final stage, which is the whole point of the split.

## Compose

```yaml
services:
  [APP_NAME]:
    image: [GITHUB_USER]/[APP_NAME]:latest
    container_name: [APP_NAME]
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=UTC
    restart: unless-stopped
```

The host directory backing `/data` is owned by UID 10001 on the host, because that is the user writing it inside the container. A `chown 10001:10001 ./data` before the first run is what keeps the container from failing to write on start.

## Health Check

Add one when something orchestrates the container and can act on the result. On a plain `docker run` it changes a status string and nothing else.

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1
```

## Placeholders

| Placeholder | Replace with |
|---|---|
| `[APP_NAME]` | the application name and binary or entry file |
| `[GITHUB_USER]` | the image namespace |
| `1.26.6`, `3.24.1`, `trixie`, `24.19.0` | the current pinned versions, checked rather than assumed |
