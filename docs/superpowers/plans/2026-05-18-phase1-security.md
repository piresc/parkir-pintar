# Phase 1: Security & Auth Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate all security vulnerabilities: exposed JWT secret in frontend, broken auth flow, timing-attack-vulnerable API key comparison, missing gateway protections, and infrastructure secret leaks.

**Architecture:** Server-side auth via existing gateway JWT middleware; remove client-side JWT signing; add constant-time comparison for API keys; add rate limiting and CORS to gateway; create `.dockerignore`; fix Terraform secrets.

**Tech Stack:** Go (crypto/subtle), React, Gin, Terraform, Docker

---

### Task 1: Remove JWT Signing from Frontend

**Files:**
- Delete: `frontend/src/utils/jwt.js`
- Modify: `frontend/src/pages/LoginPage.jsx`
- Modify: `frontend/src/contexts/AuthContext.jsx`
- Modify: `frontend/src/api/client.js`

The frontend currently imports a JWT secret and signs tokens client-side. This must be replaced with a server-side login endpoint that returns a signed token.

- [ ] **Step 1: Remove the jwt.js utility file**

Delete `frontend/src/utils/jwt.js` entirely.

- [ ] **Step 2: Update LoginPage to call a server-side login endpoint**

Replace the client-side JWT generation in `frontend/src/pages/LoginPage.jsx`:

```jsx
// Replace the existing handleLogin function (around lines 20-40)
const handleLogin = async (e) => {
  e.preventDefault();
  setError('');
  setLoading(true);

  try {
    const response = await fetch(`${import.meta.env.VITE_API_URL || ''}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ driver_id: driverId }),
    });

    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.error || 'Login failed');
    }

    const data = await response.json();
    login(data.data.token);
  } catch (err) {
    setError(err.message || 'Login failed. Please try again.');
  } finally {
    setLoading(false);
  }
};
```

- [ ] **Step 3: Add token expiration checking to AuthContext**

Modify `frontend/src/contexts/AuthContext.jsx` to check token expiry:

```jsx
import { createContext, useContext, useState, useEffect, useCallback } from 'react';

const AuthContext = createContext(null);

function decodePayload(token) {
  try {
    const base64 = token.split('.')[1];
    return JSON.parse(atob(base64));
  } catch {
    return null;
  }
}

function isTokenExpired(token) {
  const payload = decodePayload(token);
  if (!payload || !payload.exp) return true;
  return Date.now() >= payload.exp * 1000;
}

export function AuthProvider({ children }) {
  const [token, setToken] = useState(() => {
    const stored = localStorage.getItem('pp_token');
    if (stored && !isTokenExpired(stored)) return stored;
    localStorage.removeItem('pp_token');
    return null;
  });

  const user = token ? decodePayload(token) : null;

  const login = useCallback((newToken) => {
    localStorage.setItem('pp_token', newToken);
    setToken(newToken);
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('pp_token');
    setToken(null);
  }, []);

  // Periodic expiry check
  useEffect(() => {
    if (!token) return;
    const interval = setInterval(() => {
      if (isTokenExpired(token)) {
        logout();
      }
    }, 30000); // check every 30s
    return () => clearInterval(interval);
  }, [token, logout]);

  return (
    <AuthContext.Provider value={{ token, user, login, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
```

- [ ] **Step 4: Fix API client to conditionally include Authorization header**

Modify `frontend/src/api/client.js` around line 10:

```js
export async function apiRequest(path, options = {}) {
  const token = localStorage.getItem('pp_token');
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    localStorage.removeItem('pp_token');
    window.location.href = '/login';
    throw new Error('Session expired');
  }

  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || `Request failed: ${response.status}`);
  }

  return response.json();
}
```

- [ ] **Step 5: Add a login endpoint to the gateway**

Create the login handler in `internal/gateway/handler/handler.go`. Add after the existing route registrations:

```go
// In RegisterRoutes, add before the JWT-protected group:
public := engine.Group("/api/v1")
{
    public.POST("/auth/login", h.Login)
}
```

Add the handler method:

```go
// Login generates a JWT for a driver. In production this would validate
// credentials; for now it accepts a driver_id and returns a signed token.
func (h *Handler) Login(c *gin.Context) {
    var req struct {
        DriverID string `json:"driver_id" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "driver_id is required")
        return
    }

    token, err := auth.GenerateToken(req.DriverID, "driver", h.cfg.JWT.Secret)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, "failed to generate token")
        return
    }

    response.Success(c, http.StatusOK, gin.H{"token": token})
}
```

- [ ] **Step 6: Remove VITE_JWT_SECRET from all env files**

Remove any `VITE_JWT_SECRET` entries from:
- `frontend/.env` (if exists)
- `config/.env`
- `config/.env.example`
- `example.env`

- [ ] **Step 7: Verify frontend builds without jwt.js**

Run:
```bash
cd frontend && npm run build
```
Expected: Build succeeds with no import errors.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/ internal/gateway/handler/
git commit -m "fix(security): remove client-side JWT signing, add server-side login endpoint"
```

---

### Task 2: Constant-Time API Key Comparison

**Files:**
- Modify: `pkg/middleware/auth.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/middleware/middleware_test.go` (or create `pkg/middleware/auth_test.go`):

```go
func TestAPIKeyAuth_UsesConstantTimeComparison(t *testing.T) {
    // This test verifies the middleware rejects invalid keys.
    // Constant-time behavior can't be unit-tested but we verify correctness.
    mw := NewMiddleware(slog.Default())
    keys := map[string]string{"billing": "secret-key-123"}
    handler := mw.APIKeyAuth(keys)

    // Valid key
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("GET", "/", nil)
    c.Request.Header.Set("X-API-Key", "secret-key-123")
    handler(c)
    assert.Equal(t, http.StatusOK, w.Code)

    // Invalid key
    w = httptest.NewRecorder()
    c, _ = gin.CreateTestContext(w)
    c.Request = httptest.NewRequest("GET", "/", nil)
    c.Request.Header.Set("X-API-Key", "wrong-key")
    handler(c)
    assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: Run test to verify it passes with current implementation**

Run: `go test ./pkg/middleware/ -run TestAPIKeyAuth -v`

- [ ] **Step 3: Replace map lookup with constant-time comparison**

Modify `pkg/middleware/auth.go`:

```go
import (
    "crypto/subtle"
    "net/http"
    "strings"

    "parkir-pintar/pkg/auth"
    "parkir-pintar/pkg/response"

    "github.com/gin-gonic/gin"
)

// APIKeyAuth returns middleware that validates the X-API-Key header against
// a map of expected service→key pairs using constant-time comparison.
// Returns 401 on missing or invalid keys.
func (m *Middleware) APIKeyAuth(expectedKeys map[string]string) gin.HandlerFunc {
    // Pre-compute key bytes for constant-time comparison.
    type keyEntry struct {
        service string
        key     []byte
    }
    entries := make([]keyEntry, 0, len(expectedKeys))
    for svc, key := range expectedKeys {
        entries = append(entries, keyEntry{service: svc, key: []byte(key)})
    }

    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.Abort()
            response.Error(c, http.StatusUnauthorized, "missing API key")
            return
        }

        apiKeyBytes := []byte(apiKey)
        valid := false
        for _, entry := range entries {
            if len(apiKeyBytes) == len(entry.key) &&
                subtle.ConstantTimeCompare(apiKeyBytes, entry.key) == 1 {
                valid = true
                break
            }
        }

        if !valid {
            c.Abort()
            response.Error(c, http.StatusUnauthorized, "invalid API key")
            return
        }

        c.Next()
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/middleware/ -v`
Expected: All tests pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/middleware/auth.go
git commit -m "fix(security): use constant-time comparison for API key validation"
```

---

### Task 3: Add Rate Limiting and CORS to Gateway

**Files:**
- Modify: `internal/gateway/handler/handler.go`
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Add rate limiting middleware to gateway engine setup**

In `cmd/gateway/main.go`, add rate limiting to the Gin engine setup (after middleware initialization):

```go
// After creating the middleware instance, add rate limiting
engine.Use(mw.RateLimit(cfg.Server.RateLimit))
```

- [ ] **Step 2: Add CORS middleware to gateway engine**

In `cmd/gateway/main.go`, ensure CORS is configured:

```go
import "github.com/gin-contrib/cors"

// Add before route registration:
engine.Use(cors.New(cors.Config{
    AllowOrigins:     cfg.Server.CORSOrigins,
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Idempotency-Key"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

- [ ] **Step 3: Add CORS origins to config**

In `pkg/config/config.go`, add to the Server config struct:

```go
type ServerConfig struct {
    // ... existing fields ...
    CORSOrigins []string `env:"CORS_ORIGINS" envDefault:"http://localhost:5173"`
    RateLimit   RateLimitConfig
}
```

- [ ] **Step 4: Verify gateway builds**

Run: `go build ./cmd/gateway/`
Expected: Compiles successfully.

- [ ] **Step 5: Commit**

```bash
git add cmd/gateway/main.go pkg/config/config.go internal/gateway/
git commit -m "feat(gateway): add rate limiting and CORS middleware"
```

---

### Task 4: Harden Gateway Ownership Check

**Files:**
- Modify: `internal/gateway/handler/handler.go:113`

- [ ] **Step 1: Write the failing test**

Add to `internal/gateway/handler/handler_test.go`:

```go
func TestCreateReservation_RejectsEmptyUserID(t *testing.T) {
    // Simulate a request where JWT middleware somehow didn't set user_id
    router := setupTestRouter() // uses test handler
    w := httptest.NewRecorder()
    body := `{"driver_id":"driver-1","vehicle_type":"car","assignment_mode":"system_assigned","idempotency_key":"test-key"}`
    req := httptest.NewRequest("POST", "/api/v1/reservations", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    // Deliberately NOT setting user_id in context
    router.ServeHTTP(w, req)
    assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

- [ ] **Step 2: Fix the ownership check**

In `internal/gateway/handler/handler.go`, replace the ownership check pattern:

```go
// Before (around line 113):
// if uid := getUserID(c); uid != "" && req.DriverID != uid {

// After:
uid := getUserID(c)
if uid == "" {
    response.Error(c, http.StatusUnauthorized, "user identity not found")
    return
}
if req.DriverID != uid {
    response.Error(c, http.StatusForbidden, "cannot create reservation for another driver")
    return
}
```

Apply the same pattern to all endpoints that use `getUserID`.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/gateway/handler/ -v`
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/handler/
git commit -m "fix(security): reject requests with missing user identity post-auth"
```

---

### Task 5: Create .dockerignore

**Files:**
- Create: `.dockerignore`

- [ ] **Step 1: Create the .dockerignore file**

```dockerignore
# Version control
.git
.github
.gitignore

# Environment and secrets
.env
config/.env
deploy/staging/.env
infra/

# Documentation
docs/
*.md
!README.md

# Tests
tests/
**/*_test.go

# Build artifacts
bin/
.coverage/
frontend/node_modules/
frontend/dist/

# IDE
.vscode/
.idea/

# Monitoring configs (not needed in app images)
deploy/monitoring/

# Temporary
/tmp/
```

- [ ] **Step 2: Verify Docker build still works**

Run: `docker build -f build/gateway.Dockerfile -t test-gateway .`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add .dockerignore
git commit -m "fix(security): add .dockerignore to prevent secret leakage into images"
```

---

### Task 6: Remove Dev CA Certificate from Production Images

**Files:**
- Modify: `build/billing.Dockerfile`
- Modify: `build/gateway.Dockerfile`
- Modify: `build/payment.Dockerfile`
- Modify: `build/presence.Dockerfile`
- Modify: `build/reservation.Dockerfile`
- Modify: `build/search.Dockerfile`

- [ ] **Step 1: Make cert copy conditional via build arg**

In each `build/*.Dockerfile`, replace the unconditional COPY with:

```dockerfile
# Replace:
# COPY infra/certs/dev/ca.pem /etc/ssl/certs/parkir-pintar-ca.pem

# With:
ARG INCLUDE_DEV_CERTS=false
RUN if [ "$INCLUDE_DEV_CERTS" = "true" ] && [ -f /tmp/ca.pem ]; then \
      cp /tmp/ca.pem /etc/ssl/certs/parkir-pintar-ca.pem; \
    fi
```

And in the builder stage, conditionally copy:

```dockerfile
ARG INCLUDE_DEV_CERTS=false
COPY infra/certs/dev/ca.pem* /tmp/
```

- [ ] **Step 2: Update docker-compose to pass build arg for dev**

In `docker-compose.yml`, add to each service's build section:

```yaml
build:
  args:
    INCLUDE_DEV_CERTS: "true"
```

- [ ] **Step 3: Verify images build without the arg (production default)**

Run: `docker build -f build/gateway.Dockerfile -t test-gateway .`
Expected: Build succeeds, no ca.pem in image.

- [ ] **Step 4: Commit**

```bash
git add build/ docker-compose.yml
git commit -m "fix(security): make dev CA cert conditional, not included by default"
```

---

### Task 7: Fix Terraform Secret Management

**Files:**
- Modify: `infra/terraform/main.tf`

- [ ] **Step 1: Replace plaintext DATABASE_URL with Secret Manager reference**

In `infra/terraform/main.tf`, replace the env block (around line 170):

```hcl
# Replace:
# env {
#   name  = "DATABASE_URL"
#   value = "postgresql://${google_sql_user.app.name}:${var.db_password}@..."
# }

# With:
resource "google_secret_manager_secret" "database_url" {
  secret_id = "parkir-pintar-database-url"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "database_url" {
  secret      = google_secret_manager_secret.database_url.id
  secret_data = "postgresql://${google_sql_user.app.name}:${var.db_password}@${google_sql_database_instance.main.private_ip_address}:5432/${google_sql_database.main.name}?sslmode=require"
}

# In the Cloud Run service template:
env {
  name = "DATABASE_URL"
  value_source {
    secret_key_ref {
      secret  = google_secret_manager_secret.database_url.secret_id
      version = "latest"
    }
  }
}
```

- [ ] **Step 2: Add prevent_destroy to Cloud SQL**

```hcl
resource "google_sql_database_instance" "main" {
  # ... existing config ...

  lifecycle {
    prevent_destroy = true
  }
}
```

- [ ] **Step 3: Validate Terraform**

Run: `cd infra/terraform && terraform validate`
Expected: Configuration is valid.

- [ ] **Step 4: Commit**

```bash
git add infra/terraform/
git commit -m "fix(security): use Secret Manager for DB credentials, add prevent_destroy"
```

---

### Task 8: Enforce JWT Secret Minimum Length

**Files:**
- Modify: `pkg/config/config.go`
- Modify: `pkg/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add to `pkg/config/config_test.go`:

```go
func TestConfig_RejectsShortJWTSecret(t *testing.T) {
    t.Setenv("APP_ENV", "staging")
    t.Setenv("SERVER_PORT", "8080")
    t.Setenv("JWT_SECRET", "short") // Only 5 chars

    _, err := Load("") // Load without file
    require.Error(t, err)
    assert.Contains(t, err.Error(), "JWT_SECRET must be at least 32 characters")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/ -run TestConfig_RejectsShortJWTSecret -v`
Expected: FAIL (currently accepts short secrets)

- [ ] **Step 3: Add minimum length validation**

In `pkg/config/config.go`, in the `Validate()` method, after the empty check:

```go
if cfg.JWT.Secret == "" {
    return fmt.Errorf("JWT_SECRET is required")
}
if cfg.AppEnv != "local" && cfg.AppEnv != "test" && len(cfg.JWT.Secret) < 32 {
    return fmt.Errorf("JWT_SECRET must be at least 32 characters in non-local environments")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/config/ -run TestConfig_RejectsShortJWTSecret -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/config/
git commit -m "fix(security): enforce minimum 32-char JWT secret in non-local environments"
```
