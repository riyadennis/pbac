# PBAC — Policy Based Access Control

A REST API service for managing and evaluating access control policies using [Open Policy Agent (OPA)](https://github.com/open-policy-agent/opa) with PostgreSQL as the policy store.

## Overview

PBAC lets you store Rego policies in a database and evaluate them at runtime against arbitrary input. This enables dynamic, centralised access control without redeploying your application.

```
Client → POST /policies/{id}/evaluate → OPA evaluates Rego → { "allow": true/false }
```

## Architecture

```
cmd/server/main.go                  # Entry point, wiring, HTTP server
internal/
  model/policy.go                   # Domain types
  repository/policy_repository.go   # PostgreSQL CRUD (pgx/v5)
  service/policy_service.go         # Business logic, OPA evaluation
  handler/policy_handler.go         # HTTP handlers (chi router)
migrations/                         # SQL migrations (golang-migrate)
docker-compose.yml                  # Local PostgreSQL
```

## Prerequisites

- Go 1.22+
- Docker (for local PostgreSQL)

## Getting Started

### 1. Start PostgreSQL

```bash
docker-compose up -d
```

### 2. Configure environment

```bash
cp .env.example .env
```

`.env` defaults:
```
DATABASE_URL=postgres://pbac:pbac@localhost:5432/pbac?sslmode=disable
PORT=8080
```

### 3. Run the server

```bash
go run ./cmd/server
```

Database migrations are applied automatically on startup.

## API Reference

### Policy object

```json
{
  "id":          "550e8400-e29b-41d4-a716-446655440000",
  "name":        "Admin Access",
  "description": "Allow admins only",
  "module":      "authz",
  "content":     "package authz\n\ndefault allow = false\n\nallow {\n  input.role == \"admin\"\n}",
  "created_at":  "2026-03-16T10:00:00Z",
  "updated_at":  "2026-03-16T10:00:00Z"
}
```

The `module` field is the Rego package name (e.g. `authz` maps to `package authz` in your Rego content).

### Endpoints

| Method   | Path                        | Description              |
|----------|-----------------------------|--------------------------|
| `POST`   | `/policies`                 | Create a policy          |
| `GET`    | `/policies`                 | List all policies        |
| `GET`    | `/policies/{id}`            | Get a policy by ID       |
| `PUT`    | `/policies/{id}`            | Update a policy          |
| `DELETE` | `/policies/{id}`            | Delete a policy          |
| `POST`   | `/policies/{id}/evaluate`   | Evaluate a policy        |

---

### Create a policy

```
POST /policies
```

**Request body:**

```json
{
  "name":        "Admin Access",
  "description": "Allow requests from admin role only",
  "module":      "authz",
  "content":     "package authz\n\ndefault allow = false\n\nallow {\n  input.role == \"admin\"\n}"
}
```

**Response** `201 Created`:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Admin Access",
  ...
}
```

> Rego syntax is validated before the policy is stored. Invalid Rego returns `400 Bad Request`.

---

### List all policies

```
GET /policies
```

**Response** `200 OK`:

```json
[
  { "id": "...", "name": "Admin Access", ... },
  { "id": "...", "name": "Read Only", ... }
]
```

---

### Get a policy

```
GET /policies/{id}
```

**Response** `200 OK` — single policy object, or `404 Not Found`.

---

### Update a policy

```
PUT /policies/{id}
```

**Request body:** same fields as create. All fields are replaced.

**Response** `200 OK` — updated policy object.

---

### Delete a policy

```
DELETE /policies/{id}
```

**Response** `204 No Content`, or `404 Not Found`.

---

### Evaluate a policy

```
POST /policies/{id}/evaluate
```

**Request body:**

```json
{
  "input": {
    "role": "admin",
    "resource": "orders",
    "action": "delete"
  },
  "query": "data.authz.allow"
}
```

- `input` — the data passed to OPA for evaluation
- `query` — OPA query to run (defaults to `data.<module>.allow` if omitted)

**Response** `200 OK`:

```json
{
  "result": true,
  "allow": true
}
```

---

## Writing Rego Policies

Policies are written in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/). The `module` field must match the `package` declaration in your Rego content.

### Example: Role-based access

```rego
package authz

default allow = false

allow {
  input.role == "admin"
}

allow {
  input.role == "editor"
  input.action != "delete"
}
```

### Example: Resource-level permissions

```rego
package resource_policy

default allow = false

allow {
  input.user == "alice"
  input.resource == "reports"
}

allow {
  input.role == "superadmin"
}
```

Evaluate with:

```bash
curl -X POST http://localhost:8080/policies/{id}/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "input": { "user": "alice", "resource": "reports" },
    "query": "data.resource_policy.allow"
  }'
```

## Development

### Build

```bash
go build ./...
```

### Vet

```bash
go vet ./...
```

### Database schema

The `policies` table is created automatically via migrations in `migrations/`. To roll back:

```bash
# requires golang-migrate CLI
migrate -path migrations -database "$DATABASE_URL" down 1
```

## Dependencies

| Package | Purpose |
|---------|---------|
| [open-policy-agent/opa](https://github.com/open-policy-agent/opa) | Rego policy parsing and evaluation |
| [jackc/pgx/v5](https://github.com/jackc/pgx) | PostgreSQL driver and connection pool |
| [go-chi/chi/v5](https://github.com/go-chi/chi) | HTTP router |
| [golang-migrate/migrate/v4](https://github.com/golang-migrate/migrate) | Database migrations |
| [google/uuid](https://github.com/google/uuid) | UUID generation for policy IDs |
| [joho/godotenv](https://github.com/joho/godotenv) | `.env` file loading |