# cnpg.io Service Broker

An Open Service Broker implementation for [CloudNativePG.io](https://cloudnative-pg.io/)

## Features

- ✅ Provision and deprovision PostgreSQL clusters via cnpg.io
- ✅ Service binding with connection credentials
- ✅ Multiple plans and t-shirt sizes (small, medium, large)
- ✅ Plan updates (scale up)
- ✅ High availability clusters with PgBouncer pooling
- ✅ TLS certificate management
- ✅ LoadBalancer service creation
- ✅ Health checks and monitoring
- ✅ Structured logging
- ✅ HTTP BasicAuth security

## Architecture

The broker consists of several components:

- **Router**: HTTP server setup and middleware
- **Broker**: OSB API implementation
- **CNPG Client**: Kubernetes API interaction
- **Catalog**: Service and plan definitions
- **Health**: Kubernetes connectivity checks
- **Metrics**: Prometheus metrics exposition

## Requirements

- Kubernetes cluster (1.30+)
- CloudNativePG operator (1.28+)
- RBAC permissions to manage namespaces, clusters, services, secrets, poolers

## Quick Start

### Local Development

```bash
# Create local kind cluster with CNPG
make kind

# Run with hot-reload
make run

# Test endpoints
make provision
make fetch-instance
make bind-instance
make deprovision
```

### Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `BROKER_USERNAME` | BasicAuth username | (none) |
| `BROKER_PASSWORD` | BasicAuth password | (none) |
| `BROKER_LOG_LEVEL` | Log level (debug/info/warn/error) | info |
| `BROKER_LOG_TIMESTAMP` | Include timestamps in logs | false |

## API Endpoints

### OSB API (v2)

- `GET /v2/catalog` - List available services and plans
- `PUT /v2/service_instances/{instance_id}` - Provision a Postgres instance
- `PATCH /v2/service_instances/{instance_id}` - Update instance plan (scale up only)
- `GET /v2/service_instances/{instance_id}` - Get instance status
- `DELETE /v2/service_instances/{instance_id}` - Deprovision instance
- `PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}` - Create binding
- `GET /v2/service_instances/{instance_id}/service_bindings/{binding_id}` - Get binding
- `DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}` - Delete binding

### Health & Metrics

- `GET /health` or `/healthz` - Health check endpoint
- `GET /metrics` - Prometheus metrics

## Service Plans

### Development Plans (Single Instance)

- **dev-small**: 1 instance, 0.5 CPU, 512MB RAM, 10GB storage
- **dev-medium**: 1 instance, 2 CPU, 2GB RAM, 50GB storage
- **dev-large**: 1 instance, 4 CPU, 4GB RAM, 250GB storage

### Production Plans (High Availability)

- **small**: 2 instances, 1 CPU, 1GB RAM, 10GB storage
- **medium**: 3 instances, 2 CPU, 2GB RAM, 50GB storage
- **large**: 3 instances, 4 CPU, 4GB RAM, 250GB storage

HA plans include:
- Primary/standby replication
- Automatic failover
- PgBouncer connection pooling
- LoadBalancer services

### Plan Updates

Plans can be updated to scale up resources:
- **Instances**: Can only increase or stay the same (e.g., 2 → 3)
- **Storage**: Can only increase or stay the same (e.g., 10GB → 50GB)
- **CPU/Memory**: Can only increase or stay the same (e.g., 2 → 4)

Downgrades are not supported to prevent data loss.

## Credentials

Binding returns comprehensive credentials:

```json
{
  "credentials": {
    "host": "cluster-rw.namespace.svc.cluster.local",
    "port": "5432",
    "database": "app",
    "username": "app",
    "password": "...",
    "uri": "postgresql://...",
    "jdbc_uri": "jdbc:postgresql://...",
    "ro_host": "cluster-ro.namespace.svc.cluster.local",
    "ro_uri": "postgresql://...",
    "ro_jdbc_uri": "jdbc:postgresql://...",
    "lb_host": "1.2.3.4",
    "lb_uri": "postgresql://...",
    "lb_jdbc_uri": "jdbc:postgresql://...",
    "pooler_host": "1.2.3.4",
    "pooler_port": "6432",
    "pooler_uri": "postgresql://...",
    "pooler_jdbc_uri": "jdbc:postgresql://...",
    "ca_cert": "-----BEGIN CERTIFICATE-----...",
    "server_cert": "-----BEGIN CERTIFICATE-----...",
    "server_key": "-----BEGIN PRIVATE KEY-----..."
  }
}
```

## Security

- HTTP BasicAuth for API access (configurable)
- Kubernetes RBAC for resource access
- TLS certificates for database connections
- Secrets stored in Kubernetes
- Ingress-level authentication (optional)

## Monitoring

Prometheus metrics available at `/metrics`:

- HTTP request metrics (via Echo middleware)
- Go runtime metrics
- Custom application metrics (future)

## Deployment

### Basic Deployment

```bash
# Deploy broker
kubectl apply -f deploy/deployment.yaml

# Verify deployment
kubectl get pods -l app=cnpg-broker
kubectl logs -l app=cnpg-broker
```

## Development

### Project Structure

```
.
├── main.go                 # Entry point
├── pkg/
│   ├── broker/            # OSB API implementation
│   ├── catalog/           # Service catalog
│   ├── cnpg/              # Kubernetes client
│   ├── config/            # Configuration management
│   ├── health/            # Health checks
│   ├── logger/            # Logging
│   ├── metrics/           # Prometheus metrics
│   ├── router/            # HTTP router
│   └── validation/        # Input validation
├── deploy/                # Kubernetes manifests
├── catalog.yaml           # Service definitions
└── Makefile              # Development tasks
```

### Testing

```bash
# Run tests
make test

# Manual testing
make provision
make fetch-instance
make bind-instance
make fetch-binding
make delete-binding
make deprovision
```

## License

Apache License 2.0 - see [LICENSE](LICENSE)

## Contributing

Contributions welcome! Please open an issue or PR.

## Links

- [CloudNativePG Documentation](https://cloudnative-pg.io/docs/)
- [Open Service Broker API](https://www.openservicebrokerapi.org/)
- [Kubernetes Operators](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
