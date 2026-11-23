---
name: dockerize
description: Containerizing any project - multi-stage builds, choosing a base image, packaging the runtime, clearing package-manager caches, and a root image whose runtime user is chosen at launch. Use when writing or changing a Dockerfile, picking a base image, deciding between alpine and debian-slim, building an image for more than one architecture, adding a docker-compose file, or when files a container writes to a bind mount come out owned by root. Triggers on Dockerfile, docker-compose.yaml, FROM, multi-stage builds, buildx, BUILDPLATFORM, TARGETARCH, apk add, apt-get install, USER, adduser, groupadd, --user, PUID, PGID, chown, ENTRYPOINT, EXPOSE, musl, and glibc.
user-invocable: false
---

# Dockerize

**Two stages, one image per architecture from one file, a runtime packaged rather than assumed where the language allows it, caches cleared inside the layer that made them, and a root image whose user is chosen at launch.**

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

`EXPOSE` documents the port. It publishes nothing on its own; the actual mapping happens at run time.

`ENTRYPOINT` names the executable, which does not change, and `CMD` supplies the default arguments, which a `docker run` can override.

## Runtime User

The image creates no user and sets no `USER`, so it runs as root unless the launch names someone else. A user baked into the image writes to a bind mount as a UID that matches nobody on the host, which trades root-owned files for files owned by a stranger.

The launch picks the user, with both IDs. `--user "$(id -u):$(id -g)"` runs the process as whoever started it, so files it writes to a bind mount on native Linux Docker belong to that person rather than to root.

```bash
mkdir -p data
docker run -d --user "$(id -u):$(id -g)" -v "$PWD/data:/data" -p 8080:8080 [GITHUB_USER]/[APP_NAME]:latest
```

Running as a different account takes that account's numbers, and neither the host nor the image needs a `/etc/passwd` entry for them. The group is always given alongside the user, because the Dockerfile reference runs a user with no primary group under the `root` group.

The ownership problem belongs to native Docker on Linux, so a macOS host launches without `--user` and runs as root. macOS numbers its first user 501 in group 20, so the IDs always come from `id` rather than from an assumed 1000.

Everything the process writes lands under the volume or `/tmp`. Every path baked into the image is owned by root and read-only to any other UID, so a write anywhere else works as root and fails as the launching user.

`HOME` points into the volume, as `ENV HOME=/data`. A UID with no `/etc/passwd` entry has no home of its own, so a config directory, a cache, or a credential file under `~` fails without it, and as root it lands in the volume instead of being lost with the container.

A Go binary built without cgo reads `HOME` through `os.UserHomeDir` and needs nothing more. `os/user.Current()` without cgo falls back to `USER` and `HOME` and returns an error when `USER` is empty, so a binary that calls it also sets `ENV USER=[APP_NAME]`.

The host directory exists before the first run. Docker creates a missing bind-mount source itself, and the native Linux daemon runs as root, so the directory it creates belongs to root and the launching user cannot write into it.

Data a root container already wrote is handed over once before switching, because the launching user cannot write into root-owned files.

```bash
sudo chown -R "$(id -u):$(id -g)" ./data
```

## Packaging the Runtime

Where a language can produce a self-contained artifact, produce one and put it on a plain base image. The image then has one thing to keep patched instead of two, and the runtime version is a property of the build rather than of whichever tag the registry resolves today.

| Language | Self-contained artifact | Final base |
|---|---|---|
| Go | `CGO_ENABLED=0` static binary | `alpine` |
| Rust | static binary, or a glibc binary | `alpine` or `debian:*-slim` |
| Python | none in the general case | `python:*-slim` |
| Java | a jlink image, or a fat jar | `eclipse-temurin:*-jre-alpine`, or `*-jre` |

When a self-contained artifact is not possible, use the slim or alpine variant of that language's official runtime image, matching the libc the artifact was built against. A full runtime image carries build tooling the running process never uses.

## alpine or debian-slim

The deciding question is whether anything in the image is linked against glibc.

A Go binary built with `CGO_ENABLED=0` links against nothing at all. It contains its own runtime, makes syscalls directly, and runs identically on musl and glibc, so alpine is the correct base and gives the smallest image.

A compiled native extension is the opposite case. A Python wheel with a compiled extension is built against glibc, and alpine's musl cannot load it. The failure appears at run time as a missing symbol rather than at build time, which is why the choice has to be made deliberately rather than discovered.

Alpine remains viable for those cases only with a deliberate musl matrix: a musl build of the runtime plus every native dependency recompiled against musl inside an alpine builder. That is a real option, not a forbidden one, but it is a decision to take on purpose. Copying a glibc artifact into an alpine image is never one.

## Go Template

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git curl make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS TARGETARCH VERSION=dev-build

# make assets populates the tree that //go:embed static needs; drop it when the
# project has no embedded frontend.
RUN make assets && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
      -ldflags="-s -w -X 'github.com/[GITHUB_USER]/[APP_NAME]/cmd.AppVersion=${VERSION}'" \
      -o /app/[APP_NAME] .

FROM alpine:3.24.4

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/[APP_NAME] .

ENV HOME=/data
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["./[APP_NAME]"]
CMD ["serve", "-d", "/data", "-H", "0.0.0.0"]
```

`--platform=$BUILDPLATFORM` pins the builder to the machine running the build, and `GOARCH=$TARGETARCH` cross-compiles from there. Without it every non-native architecture runs the whole builder under QEMU, so `make assets` and the compile execute emulated, which is the slowest part of a multi-arch build and the part Go does not need.

`VERSION` arrives as a build argument and lands in the same `-ldflags` the Makefile uses, so an image reports the version the release cut rather than `dev-build`.

`ca-certificates` is what lets the binary make an outbound HTTPS request; without it every TLS handshake fails verification. `tzdata` is what `time.LoadLocation` reads.

## Compose

```yaml
services:
  [APP_NAME]:
    image: [GITHUB_USER]/[APP_NAME]:latest
    container_name: [APP_NAME]
    user: "${PUID:-0}:${PGID:-0}"
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - TZ=UTC
    restart: unless-stopped
```

`user:` reads both IDs from the environment and falls back to root, so one file serves every host. A Linux host puts its own IDs in a `.env` beside the compose file, which compose reads on its own, and a macOS host has no `.env` and runs as root.

```bash
printf 'PUID=%s\nPGID=%s\n' "$(id -u)" "$(id -g)" > .env
```

## Multi-arch

The Dockerfile is half of a multi-arch image. The other half is the builder that assembles the manifest, since a plain `docker build` produces one architecture whatever the file says.

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v1.2.3 \
  -t [GITHUB_USER]/[APP_NAME]:v1.2.3 --push .
```

`--push` is not optional here. buildx cannot load a multi-platform result into the local daemon, which holds one image per tag, so a multi-arch build either pushes to a registry or is thrown away.

In CI the same thing needs `docker/setup-qemu-action` before `docker/setup-buildx-action`, since the emulation is what lets a non-native stage run at all. The Go template only reaches emulation for the tiny final stage, so registering QEMU costs a second and cross-compilation does the real work.

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
| `1.27.4`, `3.24.4` | the current pinned versions, checked rather than assumed |
