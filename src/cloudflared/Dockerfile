# ===============================================================================
# Cloudflare Tunnel (cloudflared) — BAUER GROUP Edition with Health Probe
# ===============================================================================
# The official cloudflare/cloudflared image is fully distroless: its only binary
# is /usr/local/bin/cloudflared — no shell, no curl, no wget. That makes a Docker
# HEALTHCHECK impossible out of the box, because the check runs *inside* the
# container and has nothing there to probe cloudflared's /ready endpoint with.
#
# This image adds exactly ONE thing: a ~2 MB static Go health probe that queries
# cloudflared's own /ready metrics endpoint. The distroless base is preserved —
# no package manager, no shell, no extra tooling — so the attack surface stays
# minimal while Docker / Compose gain a real readiness healthcheck (and
# depends_on: condition: service_healthy starts working).
#
# Source:   https://github.com/bauer-group/CS-CloudflareTunnel
# Upstream: https://github.com/cloudflare/cloudflared
# ===============================================================================

# ---------------------------------------------------------------------------
# Build Arguments
# ---------------------------------------------------------------------------
ARG BASE_GO_IMAGE=golang
ARG BASE_GO_VERSION=1.26-alpine
ARG CLOUDFLARED_IMAGE=cloudflare/cloudflared
ARG CLOUDFLARED_VERSION=latest

# ---------------------------------------------------------------------------
# Stage 1: Build the static health probe
# ---------------------------------------------------------------------------
FROM ${BASE_GO_IMAGE}:${BASE_GO_VERSION} AS probe-builder

WORKDIR /src
COPY probe/ .

# CGO disabled → fully static binary that runs on the distroless base as-is.
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /healthprobe .

# ---------------------------------------------------------------------------
# Stage 2: Runtime (distroless cloudflared + probe)
# ---------------------------------------------------------------------------
FROM ${CLOUDFLARED_IMAGE}:${CLOUDFLARED_VERSION}

# ---------------------------------------------------------------------------
# Metadata Labels (Legacy)
# ---------------------------------------------------------------------------
LABEL vendor="BAUER GROUP"
LABEL maintainer="Karl Bauer <karl.bauer@bauer-group.com>"

# ---------------------------------------------------------------------------
# OCI Image Spec Labels
# https://github.com/opencontainers/image-spec/blob/main/annotations.md
# ---------------------------------------------------------------------------
LABEL org.opencontainers.image.title="Cloudflare Tunnel"
LABEL org.opencontainers.image.description="cloudflared with a built-in static health probe against the /ready metrics endpoint — BAUER GROUP Edition"
LABEL org.opencontainers.image.vendor="BAUER GROUP"
LABEL org.opencontainers.image.authors="Karl Bauer <karl.bauer@bauer-group.com>"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/bauer-group/CS-CloudflareTunnel"
LABEL org.opencontainers.image.url="https://github.com/bauer-group/CS-CloudflareTunnel"
LABEL org.opencontainers.image.documentation="https://github.com/bauer-group/CS-CloudflareTunnel#readme"
LABEL org.opencontainers.image.version="0.1.2"

# ---------------------------------------------------------------------------
# Health Probe (single static binary — distroless stays distroless)
# ---------------------------------------------------------------------------
COPY --from=probe-builder /healthprobe /usr/local/bin/healthprobe

# cloudflared metrics / readiness endpoint (see CMD --metrics below).
# Informational only — no host port is published by this image.
EXPOSE 2000

# ---------------------------------------------------------------------------
# Health Check
# ---------------------------------------------------------------------------
# Exec form is mandatory: the base image has no shell, so a shell-form
# HEALTHCHECK (which needs /bin/sh) would fail. The probe returns exit 0 once
# cloudflared reports /ready = HTTP 200 (>= 1 active edge connection).
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/healthprobe"]

# ENTRYPOINT is inherited from the upstream image: ["cloudflared", "--no-autoupdate"]
# We only override CMD to enable the metrics server so /ready exists for the
# healthcheck (and for Prometheus scraping within the Docker network).
CMD ["tunnel", "--metrics", "0.0.0.0:2000", "run"]
