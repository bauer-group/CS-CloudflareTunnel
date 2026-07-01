# Health Check Design

## The problem

The official `cloudflare/cloudflared` image is **fully distroless**. Its entire
filesystem under the binary directories is a single file:

```
usr/local/bin/cloudflared      ← the only executable in the image
```

No `/bin/sh`, no `curl`, no `wget`, no `busybox`, no `nc`.

A Docker `HEALTHCHECK` does **not** run on the host — Docker `exec`s the check
command *inside the container's namespace*. So a healthcheck needs some
executable **inside the image** that can probe the tunnel. With the stock image
there is nothing to probe with, which is why:

- `docker ps` shows the tunnel without a health state, and
- `depends_on: { condition: service_healthy }` cannot be used against it.

Opening cloudflared's `/ready` endpoint via `--metrics` is **necessary but not
sufficient**: the endpoint becomes reachable from *outside* (host, other
containers, Prometheus), but the tunnel container still has no client to query
its own endpoint from *inside*.

## The solution

Add **exactly one** static binary — a ~2 MB dependency-free Go probe
([`src/cloudflared/probe/main.go`](../src/cloudflared/probe/main.go)) — and keep
the distroless base otherwise untouched. No package manager, no shell, no `curl`
attack surface; just one purpose-built executable that:

1. GETs `http://127.0.0.1:2000/ready` (3 s timeout), and
2. exits `0` on HTTP `200`, `1` otherwise.

`cloudflared`'s `/ready` returns `200` once the tunnel has **at least one active
edge connection**, and `503` / connection-refused otherwise — exactly the
liveness/readiness signal we want.

### Why a Go binary and not busybox/curl

| Approach | Distroless preserved | Extra attack surface | Size |
|---|:---:|---|---|
| Switch base to Alpine + curl | ❌ | shell + package manager + curl | ~10 MB+ |
| Copy in `busybox-static` | ≈ | a full multi-call shell/util binary | ~1–2 MB |
| **Static Go probe (this repo)** | ✅ | none (single-purpose, no shell) | ~2 MB |

The Go probe is the most conservative option: it cannot be repurposed as a shell
or downloader the way `busybox`/`curl` can.

## How it fits together

The image bakes in (see [`Dockerfile`](../src/cloudflared/Dockerfile)):

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/usr/local/bin/healthprobe"]
CMD ["tunnel", "--metrics", "0.0.0.0:2000", "run"]
```

- **Exec form** (`CMD ["..."]`) is mandatory — the shell form needs `/bin/sh`,
  which does not exist here.
- `--metrics 0.0.0.0:2000` enables `/ready` for both the probe and in-network
  Prometheus scraping. No host port is published.
- `HEALTHPROBE_URL` overrides the probe target if you change the metrics address.

## Result

`cloudflare-tunnel` now behaves like every other service in a BAUER GROUP stack:
it reports `healthy` / `unhealthy`, and downstream services may gate on
`depends_on: { condition: service_healthy }`.
