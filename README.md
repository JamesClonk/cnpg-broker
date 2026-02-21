# cnpg.io Service Broker

An Open Service Broker implementation for [CloudNativePG.io](https://cloudnative-pg.io/)

## Features

- Provision and deprovision PostgreSQL clusters via cnpg.io
- Service binding with connection credentials
- Multiple plans and t-shirt sizes (small, medium, large, etc..)

## Usage

```bash
# Run locally
make dev

# Build
docker build -t cnpg-broker .

# Deploy to Kubernetes
kubectl apply -f deploy/
```

## API Endpoints

- `GET /v2/catalog` - List available services and plans
- `PUT /v2/service_instances/{instance_id}` - Provision a Postgres instance
- `DELETE /v2/service_instances/{instance_id}` - Deprovision a Postgres instance
- `PUT /v2/service_instances/{instance_id}/service_bindings/{binding_id}` - Bind instance and get service credentials
- `DELETE /v2/service_instances/{instance_id}/service_bindings/{binding_id}` - Unbind instance

## Requirements

- Kubernetes cluster with a recent version of the cnpg.io operator installed, see [install and upgrade guide](https://cloudnative-pg.io/docs/1.28/installation_upgrade)
- RBAC permissions to manage cnpg.io custom resources

## TODO

- Create 1 LB service for each cluster
- Create a pooler for each HA cluster
- Have the same LB also point to the pooler, but with another port
- Have Bindings/Credentials contain the TLS certs so connections can use them if wanted

