# --- Builder stage: install Go-based security tools and build AutoAR bot ---
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Install system packages required for building tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl build-essential cmake libpcap-dev ca-certificates \
    pkg-config libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Install external tools used by AutoAR that are called as subprocess binaries.
# NOTE: httpx and nuclei are used as Go packages (go.mod), no binary needed.
# NOTE: interlace was removed — replaced by native Go concurrency.
# TruffleHog is called as a CLI binary by githubscan.go, so it must be installed here.
# Resolved upstream revisions, not moving default branches.
ARG TRUFFLEHOG_SHA=1d87fba93556b12cec2e836849971129e8bfa770
RUN git clone https://github.com/trufflesecurity/trufflehog.git /tmp/trufflehog && \
    cd /tmp/trufflehog && git checkout --detach "${TRUFFLEHOG_SHA}" && \
    test "$(git rev-parse HEAD)" = "${TRUFFLEHOG_SHA}" && \
    go build -o /go/bin/trufflehog . && \
    cd /app && rm -rf /tmp/trufflehog && \
    go install -v github.com/d3mondev/puredns/v2@v2.1.1

# Build AutoAR main CLI and entrypoint
WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod go.sum ./

# Download dependencies (module graph only)
RUN go mod download

# Copy application source
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build uses the reviewed go.mod/go.sum verbatim; no dependency graph mutation.
RUN go mod verify

# Build main autoar binary from cmd/autoar (CGO enabled for naabu/libpcap)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/autoar ./cmd/autoar

# Build entrypoint binary (replaces docker-entrypoint.sh)
WORKDIR /app/internal/scanner/entrypoint
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/autoar-entrypoint .
WORKDIR /app

# --- Runtime stage: minimal Debian image ---
FROM debian:bookworm-slim

ENV AUTOAR_SCRIPT_PATH=/usr/local/bin/autoar \
    AUTOAR_CONFIG_FILE=/app/autoar.yaml \
    AUTOAR_RESULTS_DIR=/app/new-results

WORKDIR /app

# System deps for runtime and common tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates tini jq dnsutils libpcap0.8 \
    postgresql-client \
    python3 sqlmap nmap \
    unzip zip \
    && rm -rf /var/lib/apt/lists/*

# Copy minimal application configuration and assets (source not required at runtime)
COPY regexes/ ./regexes/
COPY templates/ ./templates/
COPY web/static/data/ ./web/static/data/
COPY autoar.sample.yaml ./
COPY env.example ./

# Review/pin third-party scanner inputs separately from Go code.
ARG NUCLEI_TEMPLATES_SHA=3c37e573c530774c51973a18e697d88042ade3ec
ARG WORDLISTS_SHA=6546017dcfd538b494991a2ac748a4a38e00e9d8
RUN git clone https://github.com/h0tak88r/nuclei_templates.git nuclei_templates && \
    git -C nuclei_templates checkout --detach "${NUCLEI_TEMPLATES_SHA}" && \
    test "$(git -C nuclei_templates rev-parse HEAD)" = "${NUCLEI_TEMPLATES_SHA}" && \
    git clone https://github.com/h0tak88r/Wordlists.git Wordlists && \
    git -C Wordlists checkout --detach "${WORDLISTS_SHA}" && \
    test "$(git -C Wordlists rev-parse HEAD)" = "${WORDLISTS_SHA}"

# Copy Go tools from builder stage
COPY --from=builder /go/bin/ /usr/local/bin/
# Copy main autoar binary
COPY --from=builder /app/autoar /usr/local/bin/autoar
# Copy entrypoint binary
COPY --from=builder /app/autoar-entrypoint /usr/local/bin/autoar-entrypoint
# Create main.sh symlink to autoar for backward compatibility
RUN ln -sf /usr/local/bin/autoar /app/main.sh && \
    chmod +x /usr/local/bin/autoar 2>/dev/null || true

# Ensure directories exist
RUN mkdir -p /app/new-results /app/nuclei_templates || true

# Permissions
RUN chmod +x /usr/local/bin/autoar-entrypoint \
    && echo "All modules are now Go-based - pure Go implementation" || true

# Add a non-root user.
# `ulimit -n 1024` first: some libc/useradd versions loop over every possible file
# descriptor up to the (build container's) nofile limit, which can busy-spin for
# minutes when that limit is huge. Capping it keeps this step fast.
RUN ulimit -n 1024 2>/dev/null || true; \
    useradd -m -u 10001 autoar && \
    chown -R autoar:autoar /app && \
    chown autoar:autoar /usr/local/bin/autoar-entrypoint
USER autoar

EXPOSE 8000

# Use tini as init for proper signal handling
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/autoar-entrypoint"]

# Basic healthcheck: verify the API server responds, not just that the process exists (#18)
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD curl -sf http://localhost:${API_PORT:-8000}/health || exit 1
