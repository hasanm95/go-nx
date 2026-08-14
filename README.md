# Gonx

A reverse proxy built from scratch in Go — inspired by Nginx's architecture (master/worker processes, `SO_REUSEPORT` port sharing, event-driven request handling) and implemented from first principles: raw sockets, syscalls, and all.

Gonx reads a YAML config file, matches incoming requests against configurable path rules, load-balances across upstream servers, and forwards/relays real HTTP traffic — with correct hop-by-hop header handling, `X-Forwarded-*` headers, and config-driven custom headers with variable substitution.

---

## How a request flows through Gonx

```
Client
  │
  ▼
Worker process (bound to the shared listen port via SO_REUSEPORT)
  │
  ▼
matcher.MatchPath
  → checks the request path against config.Server.Paths
  → exact matches beat prefix matches; longest prefix wins among prefixes
  → no match → 404
  │
  ▼
upstream.BuildSelectors (built once at startup) → selector for the matched rule
  → round-robins across that rule's upstream IDs (atomic, concurrency-safe)
  │
  ▼
upstream.Lookup
  → resolves the selected ID to its real URL from config.Server.Upstreams
  → not found → 500
  │
  ▼
proxy.Forward
  → builds a new outgoing request to the upstream (path + query preserved)
  → strips hop-by-hop headers, adds X-Forwarded-*, applies configured headers
  → upstream unreachable → 502
  │
  ▼
proxy.Relay
  → copies the upstream's status, headers (hop-by-hop stripped again), and body
  → back to the original client
```

---

## Features

- **YAML config**, parsed and validated at startup (`go-playground/validator`) — malformed or logically invalid config fails fast with a clear error.
- **Path-based routing** — exact-match and prefix-match (`/admin/*`) rules, with specificity-based precedence (not config list order).
- **Round-robin load balancing** across multiple upstreams per rule, with proven concurrency safety (race-detector clean).
- **Real multi-process architecture** — a master process spawns `workers`-many independent OS processes (not just goroutines), each sharing the listen port via `SO_REUSEPORT`. Each worker independently handles unlimited concurrent connections via Go's own goroutine scheduler.
- **Hand-built reverse proxying** (no `httputil.ReverseProxy`) — correct hop-by-hop header stripping per RFC 7230 (including headers dynamically named in a `Connection` header), `X-Forwarded-For`/`-Proto`/`-Host`, connection pooling, and context-based request cancellation.
- **Configurable custom headers**, with open-ended `$variable` substitution (currently `$ip`; new variables are a one-line addition, not a restructure).

---

## Configuration

Gonx is configured entirely via a single YAML file.

```yaml
server:
  listen: 8080
  workers: 4          # optional — defaults to runtime.NumCPU() if omitted or 0

  upstreams:
    - id: node1
      url: http://localhost:8000
    - id: node2
      url: http://localhost:8001

  paths:
    - path: /
      upstreams:
        - node1
        - node2

    - path: /admin/*      # trailing "/*" marks a prefix rule; bare paths are exact-match only
      upstreams:
        - node2

  headers:
    - key: X-Forwarded-For
      value: "$ip"         # resolved per-request; unrecognized values are sent as literal text
    - key: X-Gonx-Proxy
      value: "true"
```

### Field reference

| Field | Required | Notes |
|---|---|---|
| `server.listen` | yes | Port number, `1–65535`. |
| `server.workers` | no | Number of worker processes. `0` or omitted → number of CPU cores. |
| `server.upstreams[].id` | yes | Referenced by `paths[].upstreams`. |
| `server.upstreams[].url` | yes | Must be a valid URL. |
| `server.paths[].path` | yes | Bare path (`/`, `/admin`) = exact match. Path ending in `/*` = prefix match. |
| `server.paths[].upstreams` | yes, ≥1 | List of upstream IDs this rule load-balances across. |
| `server.headers[].key` / `.value` | no | Applied to every proxied request. `$ip` in a value is replaced with the client's IP. |

**Path matching notes:**
- A request matching multiple rules resolves by specificity: exact match wins over any prefix match; among prefix matches, the longest (most specific) prefix wins. List order in the config does **not** matter.
- A bare prefix without the trailing slash (e.g. `/admin` against a `/admin/*` rule) does **not** match — matching Nginx's own default behavior. Add an explicit rule if you want both.

---

## Running it

```
go run ./cmd --config path/to/config.yml
```

The same binary acts as both master and worker — on startup it reads `GONX_ROLE` from its environment to decide which role to take. You shouldn't need to set this yourself; the master sets it automatically when spawning workers.

| Flag | Description |
|---|---|
| `--config` | Path to the YAML config file. Required. |

---

## Development

A `Dockerfile.dev` + `docker-compose.yml` + [Air](https://github.com/air-verse/air) hot-reload setup is included for local development:

```
docker compose up --build
```

This also matters for a specific reason beyond convenience: **`SO_REUSEPORT`'s load-balancing guarantee is Linux-specific.** On Linux, the kernel hashes incoming connections and spreads them across every worker sharing the port. On macOS/BSD, the same socket option exists but doesn't load-balance the same way — in practice, all traffic goes to whichever worker bound the port most recently. Since Docker Desktop runs containers inside a real Linux VM, running Gonx this way gives you genuine Linux kernel behavior even when developing on a Mac.

---

## Testing

```
go test ./...          # full suite
go test -race ./...    # concurrency safety (round-robin selector, etc.)
```

Testing approach used throughout:
- Table-driven tests with `t.Run` subtests for named, isolated results.
- [`google/go-cmp`](https://github.com/google/go-cmp) for whole-struct comparison with readable diffs, instead of dozens of manual field checks.
- `net/http/httptest` for handler- and proxy-level tests — real HTTP semantics without a real server or port.
- A dedicated concurrency test (100 goroutines against one shared selector, `go test -race`) proving the round-robin counter's `atomic` usage is genuinely thread-safe, not just correct when called sequentially.

---

## Project structure

```
cmd/                  main.go — flag parsing, master/worker branching, process orchestration
internal/config/       YAML parsing (ParseConfig) + validation (ValidateConfig)
internal/matcher/      Path-to-rule matching (MatchPath)
internal/upstream/      Round-robin selection (Selector, BuildSelectors) + ID→URL lookup (Lookup)
internal/proxy/         Request forwarding (Forward), response relay (Relay), header handling
internal/handler/       Wires matcher → upstream → proxy into a single http.HandlerFunc
```

---

## Known limitations

- **Round-robin state is per-worker-process, not global.** Since each worker is a separate OS process (no shared memory), a rule's round-robin counter is independent per worker — not a single counter shared across all `workers`. This mirrors how real Nginx workers behave, but means the *global* distribution across many requests depends on both round-robin (per worker) and the kernel's `SO_REUSEPORT` connection spreading (across workers) acting together.
- **A path rule referencing an upstream `id` that doesn't exist in `upstreams[]` is not caught by config validation.** Struct-tag-based validation can't express "this string must match an entry in that other list" — this would need a custom cross-field validation function. Currently, such a misconfiguration would only surface at request time (`upstream.Lookup` failing, returning a `500`), not at startup.
- **`SO_REUSEPORT` does not load-balance on macOS/BSD** the way it does on Linux — see the Development section above.

---

## Acknowledgments

Architecture loosely inspired by Nginx's own design, and by a Hindi-language YouTube tutorial (Piyush Garg) that builds a similar reverse proxy in Node.js/TypeScript — Gonx follows the same core ideas but is implemented from scratch in Go, including the parts (multi-process worker model, `SO_REUSEPORT`) that don't have a direct one-to-one translation from Node's `cluster` module.