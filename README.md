# Go-Notes-Api

JWT + MySQL + Chi. Prod-oriented minimal API.
Features: login, access/refresh token, RBAC (user/admin), rate-limit, JSON log, request-id, security headers, CORS, body limit, Prometheus metrics, audit log, OpenAPI, JWT key rotation (KID), refresh token reuse detection.

## Quick Start

```bash
cp .env.example .env
docker compose up -d --build   # MySQL + API + Prometheus + Grafana
curl -i http://localhost:8080/healthz
```

```bash
register → login → create note with token
curl -s -XPOST :8080/auth/register -H 'content-type: application/json' -d '{"email":"a@b.com","password":"P@ssw0rd!"}'
ACCESS=$(curl -s -XPOST :8080/auth/login -H 'content-type: application/json' -d '{"email":"a@b.com","password":"P@ssw0rd!"}' | jq -r .access)
curl -s -XPOST :8080/notes -H "authorization: Bearer $ACCESS" -H 'content-type: application/json' -d '{"title":"t","body":"b"}'
```

## Architecture

- API: Go 1.22, Chi router.
- Database: MySQL 8.x. database/sql + go-sql-driver/mysql.
- Authorization: JWT Access (HS256) + Opaque Refresh. Key rotation with KID.
- RBAC: User, admin. Control via user roles.
- Observability: Prometheus metrics, Royal Grafana.
- Security: IP and email-based rate limiting, JSON logging, request ID, CORS, security headers, body size limits, /metrics CIDR allowlist, audit logging.

## Directory Structure
```bash
cmd/
  api/         # main service
  seed-admin/  # seeding: admin role to user
  jwtkeygen/   # KID-secret generator
internal/
  config/      # .env → Config
  db/          # OpenAndMigrate (golang-migrate / file://migrations)
  server/      # router + middleware chain
  handlers/    # health, auth, notes, admin
  repos/       # users, notes, roles, refresh_tokens, audit, metric and limiter helpers
  middleware/  # auth(KID), roles, rate-limit, security headers, CORS, body limit, recover, logger, audit, allowlist
  metrics/     # Prometheus registry + http duration
  logging/     # slog JSON logger
  openapi/     # OpenAPI serve + simple Swagger UI
migrations/  # MySQL schemas
openapi/     # openapi.yaml
tests/       # k6 smoke (load/smoke.js)
ops/#prometheus.yml
```

## Data Model (summary)
- users(id, email, password_hash)
- notes(id, user_id, title, body)
- roles(id, name) + user_roles(user_id, role_id)
- refresh_tokens(token, user_id, expires_at, used_at)
- audit_logs(id, user_id?, method, path, status, ip, rid, created_at)

Reuse detection: If a refresh token that is not used_at IS NULL is reused, all active refresh tokens are canceled.


## Authentication and RBAC
- Log in → { access, refresh }
- Access the JWT: sub, exp, iat, child in the header.
- Refresh: single token; generates new access with /auth/refresh.
- RBAC: RequireRole("user") → notes; RequireRole("admin") → admin.

## Environment Variables

```bash
DB_* or DB_DSN – MySQL DSN.

JWT_KEYS, JWT_CURRENT_KID – JWT rotation with KID. 

Example:
JWT_KEYS=key1:changeme,key2:changeme2
JWT_CURRENT_KID=key2

METRICS_ALLOW – /metrics IP allowlist.

RATE_RPS, RATE_BURST – Rate limit per IP.
```

## Tips

- GET /healthz, GET /readyz, GET /info

- POST /auth/register → {id}

- POST /auth/login → {access, refresh}

- POST /auth/refresh → {access}

- GET/POST/PUT/DELETE /notes (Bearer + user role)
#### Admin:

- GET /admin/ping (Bearer + admin)

- GET /admin/users?q=&page=&size= (Bearer + admin)

- POST /admin/users/{id}/roles body: {"action":"add|remove","role":"admin|user"}

- GET /admin/audit?from=&to=&limit=&format=csv|json


## Roles
```bash
# assign admin to user
go run ./cmd/seed-admin user@example.com
```

## JWT Key Rotation
```bash
# generate new key
go run ./cmd/jwtkeygen newkid
# append the output to JWT_KEYS and set JWT_CURRENT_KID=newkid
```

## Test
```bash
export TEST_DSN='root:pass@tcp(127.0.0.1:3306)/example?parseTime=true&charset=utf8mb4'
go test ./internal/server -run Test_AdminRoleAndAudit -v
```








