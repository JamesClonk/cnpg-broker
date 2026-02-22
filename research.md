# CNPG Service Broker - Deep Technical Analysis

## Executive Summary

This is an Open Service Broker (OSB) API implementation for CloudNativePG (CNPG), a Kubernetes operator for PostgreSQL. The broker enables automated provisioning, binding, and lifecycle management of PostgreSQL database clusters on Kubernetes through a standardized REST API.

**Project Status**: Early development stage with core functionality implemented but several TODOs remaining.

**Author**: JamesClonk <jamesclonk@jamesclonk.ch>

**License**: Apache License 2.0

---

## Architecture Overview

### Technology Stack

- **Language**: Go 1.25.6
- **Web Framework**: Echo v4 (labstack/echo)
- **Kubernetes Client**: client-go v0.35.0
- **Metrics**: Prometheus client_golang v1.23.2
- **Configuration**: YAML-based catalog
- **Container**: Alpine Linux 3.23 base image

### Core Components

The application follows a clean layered architecture:

```
main.go
  └─> router (pkg/router)
       ├─> health handler (pkg/health)
       ├─> metrics handler (pkg/metrics)
       └─> broker handler (pkg/broker)
            └─> cnpg client (pkg/cnpg)
                 └─> catalog (pkg/catalog)
```

---

## Component Deep Dive

### 1. Main Entry Point (`main.go`)

**Purpose**: Application bootstrap and server initialization

**Key Features**:
- Reads PORT from environment (defaults to 8080)
- Initializes HTTP router
- Starts Echo web server

**Configuration**:
- Environment variable: `PORT` (default: 8080)

### 2. Router (`pkg/router/router.go`)

**Purpose**: HTTP server configuration and route registration

**Echo Configuration**:
- HTTP/2 disabled
- Banner and port display enabled
- Trailing slash removal middleware
- Security headers middleware
- Static file serving from `static/` directory
- Panic recovery disabled (intentionally lets the platform handle crashes)

**Registered Routes**:
- Health endpoints: `/health`, `/healthz`
- Metrics endpoint: `/metrics`
- OSB API endpoints: `/v2/*`

### 3. Health Handler (`pkg/health/handler.go`)

**Purpose**: Kubernetes liveness/readiness probes

**Endpoints**:
- `GET /health` → Returns `{"Status": "ok"}`
- `GET /healthz` → Returns `{"Status": "ok"}`

**Note**: Currently returns static response. TODO comment indicates need for actual health checks.

### 4. Metrics Handler (`pkg/metrics/handler.go`)

**Purpose**: Prometheus metrics exposure

**Endpoint**:
- `GET /metrics` → Prometheus metrics in text format

**Implementation**: Wraps `prometheus/client_golang` promhttp.Handler()

### 5. Broker Handler (`pkg/broker/handler.go`)

**Purpose**: OSB API route registration and middleware setup

**OSB API Routes** (all under `/v2` prefix):
- `GET /catalog` → List services and plans
- `PUT /service_instances/:instance_id` → Provision instance
- `GET /service_instances/:instance_id` → Get instance status
- `DELETE /service_instances/:instance_id` → Deprovision instance
- `PUT /service_instances/:instance_id/service_bindings/:binding_id` → Create binding
- `GET /service_instances/:instance_id/service_bindings/:binding_id` → Get binding
- `DELETE /service_instances/:instance_id/service_bindings/:binding_id` → Delete binding

**Planned Features** (currently commented out):
- Request logging with configurable timestamp
- HTTP Basic Authentication
- Custom log format

### 6. Broker Logic (`pkg/broker/broker.go`)

**Purpose**: Core OSB API implementation

#### GetCatalog
Returns the service catalog loaded from `catalog.yaml`.

#### ProvisionInstance
**Process**:
1. Validates instance_id parameter
2. Parses request body (service_id, plan_id, context)
3. Calls CNPG client to create cluster
4. Returns HTTP 201 on success

**Request Body**:
```json
{
  "service_id": "uuid",
  "plan_id": "uuid",
  "context": {}
}
```

#### GetInstance
**Process**:
1. Validates instance_id
2. Queries CNPG cluster existence
3. Returns HTTP 404 if not found
4. Returns HTTP 200 if exists

**OSB Compliance**: Returns HTTP 410 (Gone) for non-existent instances as per spec.

#### DeprovisionInstance
**Process**:
1. Validates instance_id
2. Checks cluster existence (returns 404 if not found)
3. Deletes cluster via CNPG client
4. Returns HTTP 200 on success

#### BindInstance
**Process**:
1. Validates instance_id
2. Retrieves credentials from CNPG client
3. Returns HTTP 201 with credentials

**Response**:
```json
{
  "credentials": {
    "host": "...",
    "port": "5432",
    "database": "...",
    "username": "...",
    "password": "...",
    "uri": "...",
    "jdbc_uri": "...",
    "ro_host": "...",
    "ro_uri": "...",
    "ro_jdbc_uri": "...",
    "ca_cert": "...",
    "tls_cert": "...",
    "tls_key": "...",
    "lb_host": "...",
    "lb_uri": "...",
    "lb_jdbc_uri": "...",
    "pooler_host": "...",
    "pooler_port": "6432",
    "pooler_uri": "...",
    "pooler_jdbc_uri": "..."
  }
}
```

#### GetBinding
Similar to BindInstance but returns HTTP 200 instead of 201.

#### UnbindInstance
Currently returns empty HTTP 200 response (no-op implementation).

### 7. CNPG Client (`pkg/cnpg/client.go`)

**Purpose**: Kubernetes API interaction for CNPG resources

**Kubernetes Resources Managed**:
- `postgresql.cnpg.io/v1/Cluster`
- `postgresql.cnpg.io/v1/Pooler`
- `core/v1/Namespace`
- `core/v1/Service`
- `core/v1/Secret`

#### Client Initialization
**Configuration Priority**:
1. In-cluster config (when running in Kubernetes)
2. Kubeconfig from `~/.kube/config` (for local development)

**Clients**:
- `dynamic.Interface` - For CNPG custom resources
- `kubernetes.Clientset` - For core Kubernetes resources

#### CreateCluster Method

**Namespace Creation**:
Creates a dedicated namespace with the instance_id as the name.

**Cluster Creation**:
Creates a CNPG Cluster resource with specifications from the selected plan:
- Instance count (1 for dev, 2-3 for HA)
- CPU requests/limits
- Memory requests/limits
- Storage size

**LoadBalancer Service**:
Creates a LoadBalancer service named `{instance_id}-lb-rw` that:
- Exposes port 5432
- Targets primary instance (selector: `cnpg.io/instanceRole: primary`)
- Provides external access to the database

**High Availability Setup** (when instances > 1):
1. **Pooler Resource**: Creates a PgBouncer pooler with:
   - Session pooling mode
   - Same instance count as cluster
   - Read-write (rw) type
   
2. **Pooler LoadBalancer**: Creates service `{instance_id}-lb-pooler` that:
   - Exposes port 6432
   - Routes to pooler pods
   - Provides connection pooling for HA clusters

#### GetCluster Method
Retrieves cluster resource from Kubernetes API.

**TODO**: Currently returns only instance_id, should return full cluster object.

#### DeleteCluster Method
Deletes the entire namespace, which cascades to delete:
- Cluster resource
- Pooler resource (if exists)
- Services
- Secrets
- All other resources in the namespace

#### GetCredentials Method

**Secret Retrieval**:
Reads the `{instance_id}-app` secret created by CNPG operator containing:
- host
- username
- password
- dbname
- fqdn-uri
- fqdn-jdbc-uri

**Credential Enrichment**:
Adds computed connection strings:
- Read-only endpoints (`{instance_id}-ro`)
- LoadBalancer endpoints (if available)
- Pooler endpoints (if HA cluster)

**TLS Certificate Retrieval**:
Attempts to read `{instance_id}-ca` secret for:
- ca.crt
- tls.crt
- tls.key

**TODO**: Certificate collection needs fixing to gather mix of *-ca, *-server, and *-pooler certs.

### 8. Catalog (`pkg/catalog/catalog.go`)

**Purpose**: Service catalog management

**Initialization**:
Loads `catalog.yaml` at package init time using `gopkg.in/yaml.v3`.

**Data Structures**:
- `Catalog`: Root structure containing services array
- `Service`: Service definition with metadata and plans
- `ServiceMetadata`: Display information and documentation links
- `Plan`: Plan definition with pricing and resource specifications
- `PlanMetadata`: Technical specifications (instances, CPU, memory, storage, HA, SLA)

**Functions**:
- `GetCatalog()`: Returns catalog as map for JSON serialization
- `PlanSpec(planId)`: Extracts resource specifications for a plan (instances, CPU, memory, storage)

**Default Fallback**: Returns (3 instances, 4 CPU, 4Gi memory, 250Gi storage) if plan not found.

---

## Service Catalog Configuration

### Service 1: postgresql-dev-db
**ID**: `79f7fb16-c95d-4210-8930-1c758648327e`

**Purpose**: Single-instance development databases

**Plans**:
1. **dev-small** (`22cedd15-900f-4625-9f10-a57f43c64585`)
   - 1 instance, 0.5 CPU, 512MB RAM, 10GB storage
   - No HA, No SLA

2. **dev-medium** (`de7acc66-412d-41c0-bf3e-763307a86c38`)
   - 1 instance, 2 CPU, 2GB RAM, 50GB storage
   - No HA, No SLA

3. **dev-large** (`bfefc341-29a1-48e5-a6be-690f44aabbb3`)
   - 1 instance, 4 CPU, 4GB RAM, 250GB storage
   - No HA, No SLA

### Service 2: postgresql-ha-cluster
**ID**: `a651d10f-25ab-4a75-99a6-520c0abbe2ae`

**Purpose**: High-availability production database clusters

**Plans**:
1. **small** (`9098f862-fb7e-42b5-9e8c-94c49e231cc3`)
   - 2 instances, 1 CPU, 1GB RAM, 10GB storage
   - HA enabled, SLA included

2. **medium** (`31aaeae1-4716-4631-b43e-93144e689427`)
   - 3 instances, 2 CPU, 2GB RAM, 50GB storage
   - HA enabled, SLA included

3. **large** (`b870dc08-1110-4bf8-ac82-e8a9d2bdd5c7`)
   - 3 instances, 4 CPU, 4GB RAM, 250GB storage
   - HA enabled, SLA included

**Metadata**:
- Display name: CloudNativePG
- Logo: https://cloudnative-pg.io/logo/large_logo.svg
- Documentation: https://cloudnative-pg.io/docs/
- Tags: cnpg, cloudnativepg, postgres, postgresql, database, relational, rdbms

---

## Deployment Architecture

### Kubernetes Resources

**ServiceAccount**: `cnpg-broker`
- Used by broker pods to authenticate with Kubernetes API

**ClusterRole**: `cnpg-broker`
Permissions:
- Namespaces: get, list, create, delete
- postgresql.cnpg.io/clusters: get, list, create, delete

**ClusterRoleBinding**: Links ServiceAccount to ClusterRole

**Service**: `cnpg-broker`
- Type: ClusterIP
- Port: 80 → 8080 (container)
- Selector: app=cnpg-broker

**Deployment**: `cnpg-broker`
- Replicas: 1
- Image: cnpg-broker:latest
- Container port: 8080
- ServiceAccount: cnpg-broker

### Container Image

**Build Process** (multi-stage):
1. **Builder stage**: golang:1.25-alpine
   - Compiles Go binary with CGO_ENABLED=0
   
2. **Runtime stage**: alpine:3.23
   - Installs ca-certificates
   - Copies public/, static/, and binary
   - Exposes port 8080

**Image Size Optimization**: Static binary with minimal Alpine base.

---

## Development Workflow

### Local Development

**Prerequisites**:
- Go 1.25.6
- Kubernetes cluster (kind recommended)
- kubectl configured
- CNPG operator installed

**Setup Commands**:
```bash
# Initialize dependencies
make init

# Create local kind cluster with CNPG
make kind

# Run with hot-reload
make run

# Run with race detector
make dev
```

**Environment Configuration**:
- `_fixtures/env`: Base configuration (PORT=9999, credentials)
- `.env_private`: Private/local overrides (gitignored)

### Hot Reload Configuration

**Tool**: Air v1.64.5

**Configuration** (`.air.toml`):
- Watch: `.go`, `.tpl`, `.tmpl`, `.html` files
- Exclude: vendor, tmp, testdata, _fixtures, *_test.go
- Build delay: 1000ms
- Rerun delay: 500ms
- Output: `./tmp/cnpg-broker`
- Logs: `tmp/build-errors.log`

### Testing Endpoints

**Makefile targets** for manual testing:
- `make provision`: Creates test instance
- `make fetch-instance`: Queries instance status
- `make deprovision`: Deletes instance
- `make bind-instance`: Creates binding
- `make fetch-binding`: Queries binding
- `make delete-binding`: Deletes binding

**Test Credentials**: disco:dingo (from _fixtures/env)

**Test Instance ID**: fe5556b9-8478-409b-ab2b-3c95ba06c5fc

**Test Binding ID**: db59931a-70a6-43c1-8885-b0c6b1c194d4

### Kind Cluster Setup

**Cluster Name**: cnpg

**Installed Components**:
1. cert-manager v1.19.2
2. CloudNativePG operator v1.28.1
3. Barman Cloud plugin v0.11.0

**Deployment Verification**:
- Waits for cert-manager deployment
- Waits for cnpg-controller-manager deployment
- Waits for barman-cloud deployment
- Timeout: 60s per component

---

## Open Service Broker API Compliance

### Implemented Endpoints

✅ **GET /v2/catalog**
- Returns service offerings and plans
- Compliant with OSB spec

✅ **PUT /v2/service_instances/:instance_id**
- Provisions new PostgreSQL cluster
- Returns HTTP 201 on success
- Accepts service_id, plan_id, context

✅ **GET /v2/service_instances/:instance_id**
- Retrieves instance status
- Returns HTTP 404 for non-existent instances

✅ **DELETE /v2/service_instances/:instance_id**
- Deprovisions cluster
- Returns HTTP 200 on success
- Returns HTTP 404 if instance doesn't exist

✅ **PUT /v2/service_instances/:instance_id/service_bindings/:binding_id**
- Creates binding and returns credentials
- Returns HTTP 201 with full credential set

✅ **GET /v2/service_instances/:instance_id/service_bindings/:binding_id**
- Retrieves existing binding credentials
- Returns HTTP 200

✅ **DELETE /v2/service_instances/:instance_id/service_bindings/:binding_id**
- Unbinds instance (currently no-op)
- Returns HTTP 200

### Missing OSB Features

❌ **Asynchronous Operations**
- No support for long-running operations
- No `accepts_incomplete` parameter handling
- No `/last_operation` endpoint

❌ **Service Instance Parameters**
- No custom parameter support during provisioning

❌ **Binding Parameters**
- No custom parameter support during binding

❌ **Orphan Mitigation**
- No cleanup of orphaned resources

---

## Security Considerations

### Current State

**Authentication**: 
- ⚠️ Currently disabled (commented out in handler.go)
- Planned: HTTP Basic Auth with username/password from config.yaml

**Authorization**:
- No role-based access control
- All authenticated users have full access

**Kubernetes RBAC**:
- ServiceAccount with ClusterRole
- Permissions limited to namespaces and CNPG clusters
- ⚠️ Missing permissions for Services, Secrets, Poolers in ClusterRole

**Secrets Management**:
- Credentials stored in Kubernetes Secrets
- Secrets created by CNPG operator
- TLS certificates available but collection incomplete

**Network Security**:
- LoadBalancer services expose databases externally
- TLS support available (optional)

### Recommendations

1. **Enable Authentication**: Uncomment BasicAuth middleware
2. **Add RBAC**: Implement role-based access control
3. **Fix ClusterRole**: Add missing resource permissions
4. **Audit Logging**: Log all provisioning/binding operations
5. **Secret Rotation**: Implement credential rotation

---

## Known Issues and TODOs

### Code TODOs

1. **broker.go**:
   - Implement proper config system for logging and auth

2. **handler.go**:
   - Enable request logging middleware
   - Enable BasicAuth middleware
   - Implement config-driven timestamp logging

3. **client.go**:
   - Fix TLS certificate collection (mix of ca, server, pooler certs)
   - Return actual cluster object from GetCluster (not just ID)

4. **catalog.go**:
   - Replace panic() with logger.Fatal()

5. **health/handler.go**:
   - Implement actual health checks (database connectivity, operator status)

### Functional Gaps

1. **No Plan Updates**: Cannot change plan after provisioning
2. **No Async Operations**: Blocks on long-running operations
3. **No Backup Configuration**: Barman plugin installed but not configured
4. **No Monitoring Setup**: Prometheus metrics exposed but no dashboards
5. **No Validation**: Minimal input validation
6. **No Idempotency**: PUT operations not fully idempotent

### Deployment Issues

1. **ClusterRole Incomplete**: Missing permissions for Services, Secrets, Poolers
2. **No Ingress**: Service is ClusterIP only
3. **No TLS**: HTTP only (no HTTPS)
4. **No Liveness Probe**: Health endpoint not used in deployment

---

## Dependencies Analysis

### Direct Dependencies

**Web Framework**:
- `github.com/labstack/echo/v4` v4.15.0
  - High-performance HTTP router
  - Middleware support
  - Context-based request handling

**Kubernetes**:
- `k8s.io/client-go` v0.35.0 - Kubernetes API client
- `k8s.io/api` v0.35.0 - Kubernetes API types
- `k8s.io/apimachinery` v0.35.0 - API machinery

**Monitoring**:
- `github.com/prometheus/client_golang` v1.23.2
  - Metrics collection and exposition

**Utilities**:
- `github.com/google/uuid` v1.6.0 - UUID generation (imported but unused)

### Indirect Dependencies (Notable)

- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing
- `golang.org/x/crypto` v0.46.0 - Cryptographic functions
- `golang.org/x/oauth2` v0.30.0 - OAuth2 support for Kubernetes auth
- `github.com/json-iterator/go` v1.1.12 - Fast JSON parsing
- `github.com/prometheus/*` - Prometheus client libraries

### Vendor Directory

**Size**: 224 entries (including subdirectories)

**Purpose**: Ensures reproducible builds with pinned dependencies

**Notable Vendored Packages**:
- Complete Kubernetes client-go stack
- Prometheus client libraries
- Echo framework and dependencies
- YAML parsers
- Protobuf libraries

---

## Performance Characteristics

### Resource Usage

**Broker Container**:
- No resource limits defined
- Minimal memory footprint (Go binary)
- Single-threaded HTTP server (Echo default)

**Database Clusters**:
- Resources defined per plan
- Range: 0.5-4 CPU, 512MB-4GB RAM, 10GB-250GB storage

### Scalability

**Broker Scalability**:
- Stateless design (can scale horizontally)
- Currently deployed as single replica
- No shared state between instances

**Database Scalability**:
- Vertical: Controlled by plan selection
- Horizontal: HA plans use 2-3 replicas
- Connection pooling via PgBouncer for HA clusters

### Bottlenecks

1. **Synchronous Operations**: Blocks on Kubernetes API calls
2. **No Caching**: Queries Kubernetes API for every request
3. **No Connection Pooling**: Creates new K8s client per request (actually cached in struct)
4. **LoadBalancer Provisioning**: Depends on cloud provider speed

---

## CloudNativePG Integration

### Operator Version

**Target**: CloudNativePG v1.28.1

**API Version**: postgresql.cnpg.io/v1

### Resources Created

**Cluster**:
- Primary/standby architecture
- Streaming replication
- Automatic failover (HA plans)
- Storage provisioned via PVC

**Pooler** (HA only):
- PgBouncer-based connection pooling
- Session pooling mode
- Scales with cluster instances
- Separate service endpoint (port 6432)

**Secrets** (created by operator):
- `{instance_id}-app`: Application credentials
- `{instance_id}-ca`: CA certificate
- `{instance_id}-server`: Server certificate
- `{instance_id}-pooler`: Pooler certificate (HA only)

### Operator Features Not Used

- Backup/restore (Barman)
- Point-in-time recovery
- Monitoring integration
- Custom PostgreSQL configuration
- Init scripts
- Import from external databases
- Declarative database/role management
- Replica clusters
- Tablespaces

---

## Comparison with Alternatives

### vs. Kubernetes Operators

**Advantages**:
- Standardized OSB API
- Platform-agnostic interface
- Suitable for service marketplaces
- Credential management built-in

**Disadvantages**:
- Additional abstraction layer
- Less direct control
- Limited to OSB capabilities

### vs. Helm Charts

**Advantages**:
- Dynamic provisioning
- Credential retrieval
- Lifecycle management API
- Multi-tenancy support

**Disadvantages**:
- More complex architecture
- Requires broker deployment
- Additional maintenance burden

### vs. Cloud Provider Services

**Advantages**:
- Kubernetes-native
- No vendor lock-in
- Full control over infrastructure
- Cost-effective for high usage

**Disadvantages**:
- Self-managed (operational burden)
- No managed backups (yet)
- Requires Kubernetes expertise

---

## Future Enhancement Opportunities

### Short-term

1. **Complete Authentication**: Enable BasicAuth middleware
2. **Fix RBAC**: Update ClusterRole with all required permissions
3. **Health Checks**: Implement real health checking
4. **TLS Certificates**: Fix certificate collection logic
5. **Input Validation**: Add request validation
6. **Error Handling**: Improve error messages and logging

### Medium-term

1. **Async Operations**: Implement OSB async operation support
2. **Plan Updates**: Enable plan changes for existing instances
3. **Backup Configuration**: Integrate Barman for automated backups
4. **Monitoring**: Add Grafana dashboards and alerts
5. **Multi-tenancy**: Implement namespace isolation
6. **Resource Quotas**: Limit instances per user/namespace

### Long-term

1. **Custom Parameters**: Support custom PostgreSQL configurations
2. **Extensions**: Allow PostgreSQL extension installation
3. **Read Replicas**: Support additional read-only replicas
4. **Cross-region**: Multi-region cluster support
5. **Disaster Recovery**: Automated DR setup
6. **Cost Management**: Usage tracking and billing integration
7. **Self-service UI**: Web interface for database management
8. **GitOps Integration**: ArgoCD/Flux support

---

## Conclusion

This CNPG Service Broker is a well-structured, early-stage implementation that successfully bridges the Open Service Broker API with CloudNativePG. The codebase demonstrates clean architecture, proper separation of concerns, and adherence to Go best practices.

**Strengths**:
- Clean, maintainable code structure
- Proper use of Kubernetes client libraries
- Comprehensive credential management
- Support for both dev and HA configurations
- Good development tooling (Makefile, Air, Kind)

**Readiness**:
- ✅ Suitable for development/testing environments
- ⚠️ Requires hardening for production use
- ❌ Not ready for multi-tenant production without security enhancements

**Next Steps for Production**:
1. Enable and test authentication
2. Fix RBAC permissions
3. Implement comprehensive health checks
4. Add monitoring and alerting
5. Configure backup/restore
6. Load testing and performance tuning
7. Security audit
8. Documentation completion

The project provides a solid foundation for a production-ready service broker with clear paths for enhancement and maturation.
