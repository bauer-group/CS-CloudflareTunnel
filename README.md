# CS-CloudflareTunnel

> **BAUER GROUP** — `cloudflared` with a built-in Docker health check.

A thin, reusable container image that wraps the official
[`cloudflare/cloudflared`](https://github.com/cloudflare/cloudflared) tunnel with
**one** thing it is missing for container orchestration: a working
`HEALTHCHECK`.

The upstream image is fully distroless (its only binary is `cloudflared` — no
shell, no `curl`), so a Docker healthcheck is impossible out of the box. This
image adds a single ~2 MB static Go probe against cloudflared's own `/ready`
endpoint and keeps the distroless base otherwise untouched.

- ✅ Real `healthy` / `unhealthy` state in `docker ps`
- ✅ `depends_on: { condition: service_healthy }` finally works for the tunnel
- ✅ Distroless preserved — no shell, no package manager, no `curl` attack surface
- ✅ `/metrics` + `/ready` exposed for Prometheus scraping (in-network)

See [docs/healthcheck.md](docs/healthcheck.md) for the full design rationale.

---

## Image

| Registry | Reference |
|---|---|
| GHCR | `ghcr.io/bauer-group/cs-cloudflaretunnel/cloudflared:latest` |
| Docker Hub | `bauergroup/cloudflared:latest` |

Tags: `latest`, `stable`, and semver tags (`x.y.z`, `x.y`, `x`) per release.

## Usage

```yaml
services:
  cloudflared:
    image: ghcr.io/bauer-group/cs-cloudflaretunnel/cloudflared:latest
    restart: unless-stopped
    environment:
      - TUNNEL_TOKEN=${TUNNEL_TOKEN}
    # ENTRYPOINT + CMD baked in:
    #   cloudflared --no-autoupdate tunnel --metrics 0.0.0.0:2000 run
    # Built-in HEALTHCHECK probes http://127.0.0.1:2000/ready
```

A complete, runnable example lives in
[docker-compose.example.yml](docker-compose.example.yml). Copy
[.env.example](.env.example) to `.env` and set `TUNNEL_TOKEN`.

### Gating a dependent service on the tunnel

```yaml
  app:
    image: my/app
    depends_on:
      cloudflared:
        condition: service_healthy   # works because of the built-in probe
```

## How it works

The image runs cloudflared with its metrics server enabled and ships a static
probe as the healthcheck command:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/healthprobe"]
CMD ["tunnel", "--metrics", "0.0.0.0:2000", "run"]
```

`healthprobe` GETs `http://127.0.0.1:2000/ready` and exits `0` on HTTP `200`
(tunnel has ≥ 1 active edge connection), `1` otherwise.

### Configuration

| Variable | Default | Purpose |
|---|---|---|
| `TUNNEL_TOKEN` | — | Cloudflare Tunnel token (remote-managed mode). **Required.** |
| `HEALTHPROBE_URL` | `http://127.0.0.1:2000/ready` | Override the probe target if you change the metrics address. |
| `TZ` | `Etc/UTC` | Container time zone. |

## Build

```bash
docker build -t cloudflared-healthcheck ./src/cloudflared
```

Two-stage build: stage 1 compiles the static Go probe (`CGO_ENABLED=0`), stage 2
copies it onto `cloudflare/cloudflared`. Build args `CLOUDFLARED_VERSION` and
`BASE_GO_VERSION` pin the upstream and builder tags.

## Maintenance & CI

Mirrors the BAUER GROUP Container-Solution automation:

- **`docker-release.yml`** — validates compose, runs semantic-release, builds &
  pushes the image to GHCR + Docker Hub with an SBOM.
- **`check-base-images.yml`** — daily digest monitor on `cloudflare/cloudflared`
  and the Go builder; a moved digest triggers a rebuild + release so upstream
  security fixes ship automatically.
- **`docker-maintenance.yml`** — auto-merges Dependabot base-image PRs.
- **`ai-issue-summary.yml`**, **`teams-notifications.yml`** — triage & alerts.

## License

[MIT](LICENSE) © 2026 BAUER GROUP.

cloudflared itself is licensed by Cloudflare, Inc. under the Apache-2.0 license.
