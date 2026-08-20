# Plan — Auth implementation (Geldlage / assetagent)

**Status:** Implemented on main working tree (not yet git-committed as slices)  
**Stand:** 2026-08-09  
**Audience:** eng  
**Depends on:** `proposal-auth-users-production.md` (direction locked)  
**Mode:** **localhost / development only** until domain purchase  
**Out of scope:** marketing site, DigitalOcean prod deploy, magic link, passkeys, buying `geldlage.com`

---

## 0. Locked product decisions (do not reopen in commits)

| Topic | Lock |
|-------|------|
| Brand hostnames (future) | `app.geldlage.com` · `api.geldlage.com` · apex later |
| Host layout | **Split** app / api (best fit for API-owned OAuth callback + `__Host-session` on API) |
| Auth ladder | Google only → (later) magic link → (later) passkeys · **no passwords** |
| Identity | Google `sub` in `auth_identities` · own **sessions** table (opaque cookie, hash at rest) |
| Tenancy | **Household** is the data boundary · alpha 1:1 owner↔household |
| Prod data | First Google login → **empty** household |
| Local data | Existing DB = **seed** · first Google login may **claim** it (`AUTH_CLAIM_EXISTING_DATA=true`) |
| Frontend login UI | Inspired by [shadcn login blocks](https://ui.shadcn.com/blocks/login) (centered card / muted shell; Google CTA only for alpha) |

### Why split hosts (not single-host `/api` proxy)

- OAuth redirect URI stays on the API (`…/auth/google/callback`) without SPA path tricks.  
- Session cookie is API-host-scoped (`__Host-session` in prod) — SPA never touches tokens.  
- Matches how App Platform / most API+SPA deploys are modeled.  
- Local mirrors prod: `localhost:5173` ↔ `localhost:8080` with CORS + `credentials: "include"` (Vite `/api` proxy becomes secondary, not the auth path).

---

## 1. Working rules for every commit

1. **Not isolated:** each commit leaves `make build`, `go test` (touched packages), and relevant `console` tests green; `make serve` + console still runnable.  
2. **No dead endpoints:** if you add a route, wire it in `serve` and OpenAPI (or document temporary test-only helpers in `_test.go` only).  
3. **Verify section is mandatory** — run the listed commands before moving on.  
4. **Dev Google project** can be created before Commit 4; Production Google client waits for domain purchase.  
5. Prefer small vertical slices over “schema-only then giant bang.”

---

## 2. Target local topology (dev)

```text
Console  http://localhost:5173
API      http://localhost:8080
Postgres docker :5432

Google OAuth (Testing project “Geldlage Development”)
  Redirect: http://localhost:8080/auth/google/callback
```

**Env (local `.env`, not committed secrets):**

```bash
APP_ENV=development
FRONTEND_URL=http://localhost:5173
CORS_ALLOWED_ORIGINS=http://localhost:5173

GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

SESSION_COOKIE_NAME=session
SESSION_COOKIE_SECURE=false
SESSION_IDLE_HOURS=336
SESSION_ABSOLUTE_HOURS=720

# First Google login attaches existing rows to that user’s household
AUTH_CLAIM_EXISTING_DATA=true
```

Console:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

(All `fetch` / generated client: `credentials: "include"`.)

---

## 3. Commit plan

---

### Commit 1 — Auth and household schema

**Message:** `Add users, households, and session tables`

**Delivers**

- Migration `00018_auth_households.sql` (names may adjust):  
  `users`, `auth_identities`, `households`, `household_memberships`, `sessions`  
- sqlc queries + thin repository methods: create user, upsert identity, create household+owner membership, create/get/revoke session by token hash  
- No HTTP auth yet; existing API behavior unchanged

**Code touch**

- `migrations/`, `queries/auth.sql` (new), `internal/repository/`, `internal/domain/`

**Verify**

```bash
make migrate-up
make sqlc-generate
go test ./internal/repository/ -count=1 -run Auth
# or package tests covering new repos
make build
```

- Manual: `\dt` shows new tables; insert round-trip in a small repo test.

**Done when:** schema exists and is covered by at least one repository test; app still serves as today.

---

### Commit 2 — Household tenancy on money data

**Message:** `Scope money data by household_id`

**Delivers**

- Migration: nullable `household_id` → backfill → `NOT NULL` + indexes + FKs on money tables (`accounts`, `transactions`, `import_runs`, `financial_baselines`, `money_reviews`, `recurring_series`, `transfer_pairs`, `forecasts` / scenarios, `decisions`, `actions`, merchant rules that are user-specific, etc.)  
- Backfill: create a single **seed household** (`name` e.g. `Local seed`) and set all existing rows to it  
- System `categories` stay global; household-owned rows only where needed  
- All list/mutate sqlc queries filter (or set) `household_id`  
- Request context type `HouseholdID` prepared; **temporary** bridge in handlers: if no auth context, use seed household id lookup (dev continuity) so the console keeps working

**Code touch**

- `migrations/`, almost all `queries/*.sql`, repositories, services, handlers as needed for compile

**Verify**

```bash
make migrate-up
make sqlc-generate
go test ./internal/... -count=1
cd console && npm test
make build
# smoke: make serve + console — transactions/baseline still load on seed household
```

- SQL check: `SELECT household_id, COUNT(*) FROM transactions GROUP BY 1` → one seed id.

**Done when:** every money query is tenant-scoped; local UI still works via seed-household fallback.

---

### Commit 3 — Sessions, `/api/me`, logout, auth middleware

**Message:** `Add session cookies and /api/me`

**Delivers**

- Session service: issue opaque token, store **SHA-256** hash, sliding + absolute expiry, revoke  
- Cookie helpers: local `session` · HttpOnly · SameSite=Lax · Secure=false  
- `GET /api/me` → `{ user, household, membership }`  
- `POST /auth/logout` → revoke + clear cookie  
- Chi middleware: resolve session → `user_id` + `household_id` on context  
- **Enforcement policy (this commit):**  
  - `/api/me` requires session (401 otherwise)  
  - other `/api/*` still use seed fallback **if** no session (keeps console green until Commit 5)  
- OpenAPI: `Me` schema + logout; regenerate client  
- Test helper: `CreateTestSession(user)` for handler tests

**Code touch**

- `internal/service/session.go`, `internal/api/handler/auth.go`, `cmd/assetagent/serve.go`, `api/openapi.yaml`, console generated client

**Verify**

```bash
make api-generate && make api-client-generate
go test ./internal/service/ ./internal/api/handler/ -count=1 -run 'Session|Me|Logout'
make build
# manual:
#   curl -c /tmp/cj -b /tmp/cj localhost:8080/api/me   → 401
#   (test helper or temporary debug issue-session in _test only)
```

**Done when:** sessions are real and tested; `/api/me` works with a cookie; money API still usable without login (fallback).

---

### Commit 4 — Google OIDC (dev)

**Message:** `Add Google sign-in OIDC flow`

**Delivers**

- Config keys from §2  
- `GET /auth/google/start` — state + nonce + PKCE S256; store verifier server-side (short-lived table or signed cookie)  
- `GET /auth/google/callback` — validate; upsert `auth_identities` by `(google, sub)`;  
  - **New user + `AUTH_CLAIM_EXISTING_DATA=true` and seed unclaimed:** attach seed household, mark claimed  
  - **New user otherwise:** create empty household + owner membership  
  - **Returning user:** load membership  
- Set session cookie; `302` → `FRONTEND_URL` (e.g. `/` or `/login?ok=1`)  
- Reject `email_verified=false`  
- Library: `coreos/go-oidc` + `golang.org/x/oauth2` (or equivalent maintained stack)  
- **Do not** store Google access/refresh tokens

**Prerequisite (human):** Google Cloud project **Geldlage Development**, OAuth Web client, redirect `http://localhost:8080/auth/google/callback`, Testing + your Google as test user.

**Verify**

```bash
go test ./internal/service/ ./internal/api/handler/ -count=1 -run 'Google|OIDC|Auth'
# Prefer httptest with fake OIDC / stub token source for CI
make build
# manual (required once):
#   open /auth/google/start → Google → back to FRONTEND_URL with session cookie on :8080
#   curl -b cookie localhost:8080/api/me → 200 with email
```

**Done when:** real localhost Google login creates session; second login reuses user; claim-seed path verified once on your DB.

---

### Commit 5 — Require auth on money API + CORS for split local hosts

**Message:** `Require sessions on API and enable CORS for the console`

**Delivers**

- Middleware: all `/api/*` except explicitly public (if any) require valid session; remove seed fallback  
- `/health` (and auth routes) stay public  
- CORS middleware: exact `CORS_ALLOWED_ORIGINS`, `AllowCredentials=true`, needed methods/headers  
- Cookie on OAuth responses: `SameSite=Lax`  
- CLI commands (`classify`, `migrate`, …) unchanged (no browser session)

**Verify**

```bash
go test ./internal/api/handler/ -count=1
# unauthenticated:
curl -i localhost:8080/api/transactions → 401
curl -i localhost:8080/health → 200
# authenticated cookie from Google login:
curl -b cookie localhost:8080/api/transactions → 200
# browser: console with VITE_API_BASE_URL must not be fully usable yet without login UI — expect failures until Commit 6
make build
```

**Done when:** API is closed without a session; CORS allows credentialed calls from `:5173`.

---

### Commit 6 — Console login gate (shadcn-inspired)

**Message:** `Add Google login page and session-aware console`

**Delivers**

- Route `/login` inspired by [shadcn login blocks](https://ui.shadcn.com/blocks/login) (prefer simple centered layout like login-01 / login-05 shell — brand mark + single **Continue with Google** button; no password fields)  
- Google button → `window.location = `${apiBase}/auth/google/start``  
- Auth gate: `useMe()`; if 401 → redirect `/login`; if ok → app shell  
- Logout control (sidebar) → `POST /auth/logout` + credentials  
- API client: `baseUrl` from `VITE_API_BASE_URL`, **`credentials: 'include'`** on all calls  
- Keep Vite `/api` proxy optional for non-auth experiments; **default path is absolute API base** so cookies stick to `:8080`  
- Tests: login page render; gate redirects when `/api/me` 401; happy path with mocked me

**UI notes**

- Copy: German or English consistent with app (“Mit Google anmelden”)  
- No email magic link UI yet (placeholder text ok: “Weitere Optionen folgen”)

**Verify**

```bash
cd console && npm test -- --run src/pages/login src/app  # adjust paths
cd console && npm run build
# manual:
#   cold load localhost:5173 → /login
#   Google → land in app with data (claimed seed)
#   logout → /login; API calls 401
```

**Done when:** full localhost loop works without curl.

---

### Commit 7 — Hardening, OpenAPI security, fixtures

**Message:** `Harden auth sessions and document API security`

**Delivers**

- OpenAPI `securitySchemes` (cookie) + default `security` on protected routes  
- Session cleanup job or lazy purge of expired rows on read  
- Absolute expiry enforced even if sliding  
- Two-household isolation test: user A cannot read user B transactions  
- `.env.example` updated (placeholders only)  
- Short `tmp/iteration-2/runbook-auth-local.md` optional: Google console steps + verify checklist  
- Production cookie name/flags gated on `APP_ENV=production` (`__Host-session`, Secure) — tested via unit tests on cookie options, not live HTTPS yet

**Verify**

```bash
make api-generate && make api-client-generate
go test ./internal/... -count=1
cd console && npm test
make build
# isolation test must be in go test suite
```

**Done when:** CI-able suite proves tenancy + session rules; docs/env examples match locked decisions.

---

## 4. Explicitly later (not in this plan)

| Item | When |
|------|------|
| Buy `geldlage.com` + DNS | After Commit 7 green |
| Google **Production** OAuth client | After DNS → `https://api.geldlage.com/auth/google/callback` |
| DigitalOcean App Platform + managed Postgres | After prod OAuth client |
| Magic link | Public beta |
| Passkeys / Apple | Post-product-market fit |
| Marketing `geldlage.com` | Separate track |
| Household invites / second member | After alpha |

---

## 5. Suggested Google setup (before Commit 4)

1. Cloud Console → new project **Geldlage Development**  
2. OAuth consent screen: External · Testing · app name Geldlage · scopes `openid email profile`  
3. Credentials → OAuth client ID → **Web application**  
4. Authorized redirect URI: `http://localhost:8080/auth/google/callback`  
5. Add your Google account as test user  
6. Put client id/secret in local `.env` only  

Exact redirect matching matters ([Google web server OAuth](https://developers.google.com/identity/protocols/oauth2/web-server)).

---

## 6. Risk watch during implementation

| Risk | What to do in-plan |
|------|--------------------|
| Giant Commit 2 | Prefer generating sqlc in one commit but keep handler changes compiling; don’t ship unscoped queries |
| Cookie + Vite proxy confusion | Commit 6 uses absolute `VITE_API_BASE_URL`; document “do not rely on proxy for auth” |
| Claim seed twice | Persist `households.claimed_at` or `bootstrap_claimed` flag so second Google user gets empty household |
| Accidental prod empty claim flag | `AUTH_CLAIM_EXISTING_DATA` defaults **false**; only local `.env` sets true |

---

## 7. Definition of done (local alpha)

- [ ] Google login on localhost works end-to-end  
- [ ] Logout clears access  
- [ ] Unauthenticated API → 401  
- [ ] Seed data visible only to the claiming Google user  
- [ ] Second Google test user sees **empty** household (no seed leak)  
- [ ] `go test ./internal/...` and console tests green  
- [ ] No secrets committed  

---

## Changelog

| Date | Change |
|------|--------|
| 2026-08-09 | Initial commit plan from locked proposal decisions |
