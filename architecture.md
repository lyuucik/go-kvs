# Go-Cloud Architecture

## Overview

Pet-проект: HTTP Key-Value Store на Go, развёрнутый в Kubernetes через GitOps. Цель — показать понимание полного DevOps-цикла: от кода приложения до observability в production-like окружении.

```
┌──────────┐     ┌──────────┐     ┌──────────────┐
│  Client   │────►│ Ingress  │────►│ go-cloud-svc │
│(curl/k6)  │     │ (nginx)  │     │  (ClusterIP) │
└──────────┘     └──────────┘     └──────┬───────┘
                                         │
                          ┌──────────────┼──────────────┐
                          ▼              ▼              ▼
                    ┌──────────┐   ┌────────┐   ┌────────────┐
                    │PostgreSQL│   │ Redis  │   │ Prometheus │
                    │ :5432    │   │ :6379  │   │ :9090      │
                    └──────────┘   └────────┘   └─────┬──────┘
                                                      │
                                                      ▼
                                                ┌──────────┐
                                                │  Grafana  │
                                                │ :3000     │
                                                └──────────┘

                    ┌──────────┐   ┌──────────────┐
                    │   Loki   │◄──│   Promtail    │
                    │ :3100    │   │ (DaemonSet)   │
                    └──────────┘   └──────────────┘
```

## Application Layer (Go)

### Структура

```
src/
  cmd/api/main.go          — точка входа: graceful shutdown, retry подключения к БД
  internal/
    api/
      handler/             — HTTP handlers (stdlib ServeMux, Go 1.22+ routing)
      metrics/             — homegrown Prometheus /metrics (atomic counters)
    kvs/
      kvs.go               — KeyValueStore interface
      store.go             — in-memory реализация (sync.RWMutex)
      postgres.go          — PostgreSQL (pgx/v5)
      cached.go            — Redis cache decorator + singleflight
      errors.go            — domain errors
```

### Ключевые решения

- **Чистая архитектура**: `KeyValueStore` interface → три реализации (in-memory, Postgres, Cached). CachedStore декорирует любую нижележащую имплементацию.
- **Cache-aside + singleflight**: при GET сначала проверяем Redis, при промахе — `singleflight.Group` гарантирует, что в БД сходит только один concurrent запрос (защита от thundering herd).
- **Cache invalidation**: PUT/DELETE сбрасывают ключ в Redis (write-through не используем — меньше нагрузки на запись).
- **Graceful shutdown**: `signal.NotifyContext` + `http.Server.Shutdown` с таймаутом 10s.
- **Healthcheck**: Docker HEALTHCHECK через wget + k8s liveness/readiness probes.
- **Без фреймворков**: stdlib http, pgx, go-redis — минимум зависимостей.

## Infrastructure Layer (Kubernetes)

### Helm Chart

```
chart/go-cloud/
  Chart.yaml               — версия 0.1.0, без external dependencies
  values.yaml              — единый конфиг: образы, ресурсы, флаги включения
  templates/
    _helpers.tpl           — common labels, fullname, namespace
    deployment.yaml        — app Deployment + probes + securityContext
    service.yaml           — app ClusterIP
    ingress.yaml           — nginx Ingress на localhost:80
    secret.yaml            — postgres пароль (plain-text, dev only)
    postgres-sts.yaml      — StatefulSet с опциональным PVC
    postgres-svc.yaml      — ClusterIP
    redis-deployment.yaml
    redis-svc.yaml
    prometheus-config.yaml — prometheus.yml с targets: go-cloud-svc:80
    prometheus-deployment.yaml
    grafana-datasource-config.yaml
    grafana-dashboard-config.yaml
    grafana-deployment.yaml
    loki-config.yaml       — single-binary, filesystem storage
    loki-sts.yaml
    loki-svc.yaml
    promtail-config.yaml   — static_configs + pipeline_stages
    promtail-ds.yaml       — DaemonSet + ClusterRole + ClusterRoleBinding
    tests/k6-test.yaml     — Helm test hook
```

Все компоненты включаются/выключаются флагами в `values.yaml`. Никаких external Helm dependencies — все образы официальные (postgres:16-alpine, redis:7-alpine, prom/prometheus, grafana/grafana, grafana/loki, grafana/promtail).

### SecurityContext

- App: `runAsUser: 1001`, `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities: drop: [ALL]`
- Postgres: `fsGroup: 999` (образ postgres требует root на старте)
- Redis/Prometheus/Grafana/Loki/Promtail — без явного securityContext (образы предусматривают свои UID)

### argoCD GitOps

```
.gitops/
  project.yaml      — AppProject: разрешённые источники, namespace, cluster-scoped ресурсы
  application.yaml  — Application: source=chart/go-cloud, destination=go-cloud, auto-sync + prune + self-heal
```

CI пушит изменение tag → ArgoCD (poll 3min) → Helm template → apply diff.

## CI/CD (GitHub Actions)

```yaml
on: push to dev/main
  job test:
    - go test -v -race ./...
  job build (needs: test):
    - docker build & push → ghcr.io/lyuucik/go-kvs:${{ github.sha }}
    - sed -i 's|^  tag: .*|  tag: $SHA|' chart/go-cloud/values.yaml
    - git commit & push
```

- `concurrency.cancel-in-progress: true` — новый коммит отменяет текущий билд.
- `if: ${{ github.actor != 'github-actions[bot]' }}` — CI не запускается на коммитах от самого себя (бесконечный цикл).

## Observability

### Метрики (Prometheus + Grafana)

Homegrown `/metrics` endpoint с атомарными счётчиками:
- `http_requests_total{code="2xx|4xx|5xx"}` — с разбивкой по статусам
- `http_request_duration_seconds_bucket{le="..."}` — latency histogram
- `kv_cache_hits_total` / `kv_cache_misses_total` / `kv_cache_errors_total`
- `kv_uptime_seconds`

Grafana dashboard provisioned через ConfigMap.

### Логи (Loki + Promtail)

- Promtail DaemonSet читает `/var/log/pods/*/*/*.log`
- pipeline_stages: regex из filename → labels (namespace, pod, container)
- Loki single-binary, filesystem storage (no S3, dev only)
- Grafana datasource auto-added

### Load Testing (k6)

Helm test hook: `helm test go-cloud -n go-cloud` запускает Job с k6:
- Smoke test (healthz, readyz, metrics, CRUD cycle)
- Mixed load: 20% PUT, 70% GET, 10% DELETE
- Пороги: `http_req_failed < 35%`, `p(95) latency < 500ms`

## Data Flow

```
PUT /api/v1/key/foo "bar"
  1. body limit 10KB
  2. store.Put → CachedStore.Put → PostgresStore.Put (SQL INSERT ... ON CONFLICT)
  3. cache invalidation: Redis DEL foo

GET /api/v1/key/foo
  1. Redis GET foo
  2. HIT → return
  3. MISS → singleflight.Do("foo") → PostgresStore.Get → Redis SET foo TTL=300 → return

DELETE /api/v1/key/foo
  1. store.Delete → PostgresStore.Delete (SQL DELETE)
  2. cache invalidation: Redis DEL foo
```

## Strengths (для собеседования)

| Аспект | Что показывает |
|--------|----------------|
| **Go + k8s** | Умение написать приложение и упаковать его в Helm |
| **GitOps** | ArgoCD, sync policy, AppProject с кластерными ресурсами |
| **CI/CD** | GitHub Actions: test → build → push → commit tag → loop protection |
| **Observability** | 3 pillars: metrics (Prometheus), logs (Loki), traces (нет, см. ниже) |
| **Caching** | Cache-aside + singleflight для thundering herd |
| **Security** | Non-root, readOnlyRootFilesystem, drop caps, body limit |
| **Helm без зависимостей** | Все образы официальные, флаги enabled/disabled |
| **k6** | Load testing как Helm test, пороговые проверки |

## Weaknesses / Что улучшить

| Проблема | Почему так | Как исправить |
|----------|------------|---------------|
| **Пароль в plain-text** | Dev-only | External Secrets / Vault / sealed-secrets |
| **Postgres без PVC** | Docker Desktop | Включить persistence.enabled в production |
| **Нет rate limiting** | Не было задачи | nginx rate limit аннотация или envoy |
| **Нет distributed tracing** | Сложно без vendor | OpenTelemetry SDK + collector + Jaeger/Tempo |
| **Grafana на ClusterIP** | Dev-only | Ingress subpath (/grafana/) + GF_SERVER_ROOT_URL |
| **Prometheus без persistent storage** | Dev-only | StatefulSet + PVC |
| **Loki filesystem storage** | Dev-only | S3/GCS backend в production |
| **Homegrown metrics** | Минимум зависимостей | prometheus/client_golang для production |
| **Один реплика app/БД** | Dev | HPA, read replicas для Postgres |
| **No canary/blue-green** | Нет трафика | Argo Rollouts + progressive delivery |
| **CI пушит в ту же ветку** | Простота | Git tag или отдельная ветка для manifests |
| **Helm test удаляется вручную** | Helm test не auto-clean | post-delete hook или внешний runner |

## Design Decisions

1. **Homegrown metrics вместо prometheus/client_golang** — 0 external dependencies для core-функции, полный контроль над форматом. В production стоит перейти на клиентскую библиотеку (регистр, гистограммы, хитрые aggregation).

2. **static_configs вместо kubernetes_sd_configs в Promtail** — `kubernetes_sd_configs` не конструирует `__path__` автоматически в Promtail 3.x, а ручная склейка source_labels ненадёжна. `static_configs` + pipeline_stages с regex из `filename`label даёт стабильный результат.

3. **Single-binary Loki вместо microservices** — для dev один pod проще. В production: distributor → ingester → querier → compactor.

4. **Plain Secret вместо External Secrets** — dev-окружение без доступа к cloud APIs. В production: либо sealed-secrets (шифрование в Git), либо External Secrets Operator (AWS Secrets Manager / GCP Secret Manager).

5. **ArgoCD sync на ту же ветку, что и CI** — CI коммитит tag, ArgoCD подхватывает. Минус: лишние коммиты в истории. Альтернатива: CI пушит tag в Git, ArgoCD читает tag из git-semver.
