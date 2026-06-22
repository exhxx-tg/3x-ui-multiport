# Monitoring API

Base path: `/panel/api/monitor`

## Health Status

```http
GET /panel/api/monitor/health
```

Returns health status for all registered protocols.

## Metrics

### Protocol Metrics
```http
GET /panel/api/monitor/metrics/:protocolId
```

Returns traffic stats, connections, uptime, errors for a specific protocol.

### All Metrics
```http
GET /panel/api/monitor/metrics
```

## Alert Rules

### List Rules
```http
GET /panel/api/monitor/rules
```

### Create Rule
```http
POST /panel/api/monitor/rules
```

### Update Rule
```http
PUT /panel/api/monitor/rules/:dbId
```

### Delete Rule
```http
DELETE /panel/api/monitor/rules/:dbId
```

## Alert History

```http
GET /panel/api/monitor/history
```

## Notifiers

### List Notifiers
```http
GET /panel/api/notifiers
```

### Test Notifier
```http
POST /panel/api/notifiers/:type/test
```

## Prometheus Metrics

```http
GET /metrics
```

Standard Prometheus metrics endpoint exposing protocol health, traffic, connections, and uptime.

## Health Checks

```http
GET /healthz
```

Kubernetes liveness probe. Returns `{"status": "ok"}` or 503.

```http
GET /readyz
```

Kubernetes readiness probe. Returns `{"status": "ready"}` or 503.
