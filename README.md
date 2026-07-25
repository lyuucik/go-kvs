# Go-Cloud

In-memory / PostgreSQL-backed Key-Value Store HTTP API with Redis cache, Prometheus metrics, and Grafana dashboards. Runs on Kubernetes with Ingress. Managed via ArgoCD GitOps.

## Architecture

```
Client ──► Ingress ──► go-cloud-svc ──► go-cloud (Go app)
                              │
                    ┌─────────┼─────────┐
                    ▼         ▼         ▼
                PostgreSQL   Redis   Prometheus
                              │           │
                              │           ▼
                              │        Grafana
                              │
                          (cache)
```

**Go app layers:**
```
http.Handler ──► keyValueHandler ──► CachedStore ──► PostgresStore
                                        │                   │
                                    Redis DEL             SQL
                                  (invalidation)     (persistence)
```

## Quick Start

### Prerequisites
- Docker Desktop with Kubernetes enabled
- `kubectl` configured
- Go 1.24+ (for local development)

### Deploy

```bash
# Build the Docker image
make build

# Deploy everything to the go-cloud namespace
make deploy

# Check status
kubectl get pods -n go-cloud
```

### Test the API

```bash
# Through Ingress
curl http://localhost/healthz
curl http://localhost/readyz

# CRUD
curl -X PUT -d "myvalue" http://localhost/api/v1/key/mykey
curl http://localhost/api/v1/key/mykey
curl -X DELETE http://localhost/api/v1/key/mykey

# Metrics
curl http://localhost/metrics
```

### Clean up

```bash
make clean
```

## API Reference

### `GET /healthz`
Liveness probe. Returns `200 OK` with body `ok`.

### `GET /readyz`
Readiness probe. Checks PostgreSQL and Redis connectivity. Returns `200 OK` or `503 Service Unavailable`.

### `PUT /api/v1/key/{key}`
Create or update a key.

**Request body:** plain text value

**Response:** `201 Created`

### `GET /api/v1/key/{key}`
Retrieve a value by key.

**Response:** `200 OK` with JSON body `{"value": "..."}`

**Errors:** `404 Not Found` if key does not exist

### `DELETE /api/v1/key/{key}`
Delete a key.

**Response:** `204 No Content`

### `GET /metrics`
Prometheus metrics in text format.

**Metrics:**
- `http_requests_total` — total request count
- `kv_cache_hits_total` — Redis cache hits
- `kv_cache_misses_total` — Redis cache misses (also includes non-existent keys)
- `kv_uptime_seconds` — application uptime

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string (if unset, uses in-memory store) |
| `REDIS_URL` | — | Redis connection string (if unset, runs without cache) |
| `CACHE_TTL` | `300` | Redis cache TTL in seconds |

### Examples

```bash
# In-memory only (no persistence)
go run ./src/cmd/api/main.go

# PostgreSQL only
DATABASE_URL=postgres://postgres:pass@localhost:5432/kvs go run ./src/cmd/api/main.go

# PostgreSQL + Redis cache
DATABASE_URL=postgres://postgres:pass@localhost:5432/kvs \
REDIS_URL=redis://localhost:6379/0 \
CACHE_TTL=300 \
go run ./src/cmd/api/main.go
```

## Development

```bash
# Run tests
make test

# Run locally (in-memory mode)
make run

# Run with PostgreSQL and Redis
docker run -d --name pg -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=kvs postgres:16
docker run -d --name redis redis:7
DATABASE_URL=postgres://postgres:pass@localhost:5432/kvs \
REDIS_URL=redis://localhost:6379/0 \
go run ./src/cmd/api/main.go
```

## Deployment

### Helm Chart

All resources are packaged as a Helm chart in `chart/go-cloud/`.

| Template | Resource | Switched by |
|----------|----------|-------------|
| `secret.yaml` | Secret | `postgres.enabled` |
| `postgres-sts.yaml` | StatefulSet | `postgres.enabled` |
| `postgres-svc.yaml` | Service | `postgres.enabled` |
| `redis-deployment.yaml` | Deployment | `redis.enabled` |
| `redis-svc.yaml` | Service | `redis.enabled` |
| `deployment.yaml` | Deployment (app) | always |
| `service.yaml` | Service (app) | always |
| `ingress.yaml` | Ingress | `ingress.enabled` |
| `prometheus-config.yaml` | ConfigMap | `monitoring.prometheus.enabled` |
| `prometheus-deployment.yaml` | Deployment + Service | `monitoring.prometheus.enabled` |
| `grafana-datasource-config.yaml` | ConfigMap | `monitoring.grafana.enabled` |
| `grafana-dashboard-config.yaml` | ConfigMap (dashboard JSON) | `monitoring.grafana.enabled` |
| `grafana-deployment.yaml` | Deployment + Service | `monitoring.grafana.enabled` |
| `tests/k6-test.yaml` | ConfigMap + Job (helm test) | `helm test` |

**Condition (dev only):** PostgreSQL runs without PersistentVolumeClaim. Data is lost on pod restart.

```bash
# Build + Install
make deploy

# Upgrade (after values.yaml changes)
make helm-upgrade

# Uninstall
make clean

# Deploy with custom values
helm upgrade --install go-cloud chart/go-cloud -f my-values.yaml
```

## Monitoring

| Component | Access | Login |
|-----------|--------|-------|
| Prometheus | ClusterIP: `prometheus:9090` | — |
| Grafana | `localhost:3000` (LoadBalancer) | `admin / admin` |

**Dashboard panels:**
1. **HTTP Requests Rate** — `rate(http_requests_total[1m])`
2. **Cache Hit / Miss Ratio** — `rate(kv_cache_hits_total[1m])` vs `rate(kv_cache_misses_total[1m])`
3. **Uptime** — `kv_uptime_seconds`

### Provisioning

All dashboards are stored as code in `deployment/monitoring/`:
- `grafana-datasource-config.yaml` — Prometheus datasource
- `grafana-dashboard-config.yaml` — Dashboard JSON via ConfigMap
- `prometheus-config.yaml` — Prometheus scrape config

## Load Testing

### In-cluster (k6 Job)

```bash
make k6-run
```

This creates a ConfigMap with the test script and a Job that runs k6 against the internal service. Results are printed to stdout and the job is cleaned up automatically.

### Local (via Docker)

```bash
# Against Ingress on host
docker run --rm -v "$(pwd):/scripts" \
  -e BASE_URL=http://host.docker.internal:80 \
  grafana/k6:latest run --duration 30s --vus 10 /scripts/test-load.js
```

### Test scenarios

The script (`test-load.js`) includes:
- **Smoke test** — validates healthz, readyz, metrics, and a full CRUD cycle
- **Mixed CRUD load** — 20% PUT, 70% GET, 10% DELETE with latency tracking and cache hit rate

## Project Structure

```
go-cloud/
├── Dockerfile                     # Multi-stage build (golang:1.25 → alpine:3.18)
├── .dockerignore
├── Makefile                       # build, deploy, test, clean
├── test-load.js                   # k6 load test scenarios (standalone)
├── go.mod / go.sum
├── src/
│   ├── cmd/api/main.go            # Entry point: graceful shutdown, probes
│   └── internal/
│       ├── api/
│       │   ├── handler/           # HTTP handlers (stdlib mux)
│       │   └── metrics/           # Custom /metrics endpoint
│       └── kvs/
│           ├── kvs.go             # KeyValueStore interface
│           ├── store.go           # In-memory implementation
│           ├── postgres.go        # PostgreSQL persistence (pgx)
│           ├── cached.go          # Redis cache decorator
│           ├── errors.go
│           ├── kvs_test.go        # Store tests
│           └── handler_test.go    # API tests
└── chart/
    └── go-cloud/                  # Helm chart
        ├── Chart.yaml
        ├── values.yaml            # All configuration parameters
        ├── .helmignore
        └── templates/
            ├── _helpers.tpl       # Template helpers (labels, names)
            ├── NOTES.txt          # Post-install instructions
            ├── secret.yaml        # PostgreSQL password Secret
            ├── postgres-sts.yaml  # PostgreSQL StatefulSet
            ├── postgres-svc.yaml  # PostgreSQL Service
            ├── redis-deployment.yaml
            ├── redis-svc.yaml
            ├── deployment.yaml    # App Deployment
            ├── service.yaml       # App Service
            ├── ingress.yaml       # Ingress (nginx)
            ├── prometheus-config.yaml
            ├── prometheus-deployment.yaml
            ├── grafana-datasource-config.yaml
            ├── grafana-dashboard-config.yaml
            ├── grafana-deployment.yaml
            └── tests/
                └── k6-test.yaml   # Helm test: k6 load test
```

## Commit History

```
4ddcfae chore: move k8s resources to go-cloud namespace, add k6 job
8d3de87 feat: add k6 load test scenarios
a08e05a chore: update Makefile with build, deploy, test, clean targets
b0f8a4f feat: add prometheus and grafana with dashboard configmap
b063b88 feat: add app deployment with retry, service and ingress
a65b939 feat: add postgres and redis manifests
81cef3f fix: correct dockerfile with multi-stage build and healthcheck
24ad8e4 feat: add prometheus metrics endpoint
7a4f2cd feat: add redis cache layer with invalidation
329ac88 feat: add postgres storage
5832ed7 refactor: migrate to stdlib, add graceful shutdown and healthcheck
```

Each commit is self-contained and individually revertible.

## Conditional Notes (Dev Environment)

- PostgreSQL runs without PersistentVolume — data is lost on pod restart
- Passwords are stored in plain-text Secrets (base64 only)
- Prometheus runs without persistent storage
- Ingress operates over HTTP without TLS
