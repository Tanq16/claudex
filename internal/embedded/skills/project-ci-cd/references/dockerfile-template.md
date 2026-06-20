# Dockerfile Template

**Applies to: CLI + Web projects only.** CLI Only projects do not have a Dockerfile.

Two-stage Docker build for Go projects with minimal runtime image.

## Full Template (With Frontend Assets)

For projects with embedded frontend that require asset downloads during build:

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git curl make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Download assets and build
RUN make assets && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o [APP_NAME] .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/[APP_NAME] .

EXPOSE 8080
ENTRYPOINT ["./[APP_NAME]"]
CMD ["-d", "/data", "-H", "0.0.0.0"]
```

---

## Minimal Template (No Frontend Assets)

For CLI + Web projects without downloadable frontend assets (NOT for CLI Only projects — those have no Dockerfile at all):

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o [APP_NAME] .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/[APP_NAME] .

ENTRYPOINT ["./[APP_NAME]"]
```

---

## Customization Notes

### Placeholders to Replace

| Placeholder | Replace With |
|-------------|--------------|
| `[APP_NAME]` | Your application name (e.g., `kairo`, `backsync`) |

### Build Stage

**Base image:** `golang:1.26-alpine`
- Alpine-based for smaller image size
- Update version as Go releases new versions

**Build dependencies:**
- `git` - Always needed for `go mod download` with VCS dependencies
- `curl` - Only needed if downloading assets
- `make` - Only needed if using Makefile during build

**Build flags:**
- `CGO_ENABLED=0` - Static binary, no C dependencies
- `GOOS=linux` - Target Linux for container runtime
- `-ldflags="-s -w"` - Strip debug info for smaller binary

### Runtime Stage

**Base image:** `alpine:latest`
- Minimal footprint (~5MB base)
- Security-focused, regularly updated

**Runtime packages:**
- `ca-certificates` - Required for HTTPS connections
- `tzdata` - Required for timezone support (time.LoadLocation)

**Ports:**
- `EXPOSE 8080` - Document the default port (customize as needed)
- This is documentation only; actual port mapping happens at runtime

### Entrypoint vs CMD

**ENTRYPOINT** - The executable, should not change:
```dockerfile
ENTRYPOINT ["./[APP_NAME]"]
```

**CMD** - Default arguments, can be overridden at runtime:
```dockerfile
# For web servers with data directory:
CMD ["-d", "/data", "-H", "0.0.0.0"]

# For simple servers:
CMD ["serve"]

# For CLI tools (no default args):
# Omit CMD entirely
```

### Common Patterns

**Adding configuration file:**
```dockerfile
COPY --from=builder /app/config.yaml .
```

**Adding data directory:**
```dockerfile
RUN mkdir -p /data
VOLUME ["/data"]
```

**Non-root user (security hardening):**
```dockerfile
RUN adduser -D -u 1000 appuser
USER appuser
```

**Health check:**
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
```

---

## docker-compose.yaml Template

For easy local deployment:

```yaml
services:
  [APP_NAME]:
    image: [GITHUB_USER]/[APP_NAME]:latest
    container_name: [APP_NAME]
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### With Environment Variables

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
      - TZ=America/New_York
    restart: unless-stopped
```
