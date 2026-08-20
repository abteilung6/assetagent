# Proposal — Auth, users, and production cutover (Geldlage)

**Status:** Direction locked — see `plan-auth-implementation.md` for commits  
**Stand:** 2026-08-09  
**Audience:** product / eng / ops  
**Surface:** production blocker before DigitalOcean + public URLs  
**Depends on:** current single-tenant assetagent API (no auth today)  
**Related:** `plan-auth-implementation.md` (commit plan); product vision cloud beta  

---

## 0. Why this exists

assetagent is a **local-first, open API, single-household DB**. Bringing it to production under a brand (working name **Geldlage**) requires identity before any internet-facing deploy.

This document:

1. Locks a **phased auth product** (Google → magic link → passkeys later).
2. Defines a **user / household** concept that fits today’s data model.
3. Validates and refines the proposed **backend-owned Google OIDC + session cookie** flow.
4. Maps **domains, cookies, CORS, DigitalOcean** without pretending the domain is bought yet.
5. Lists **open decisions** — nothing here is 100% final.

---

## 1. Current codebase reality

| Area | Today |
|------|--------|
| Auth | **None** — OpenAPI has no `securitySchemes`; `serve` mounts routes with no middleware |
| Tenancy | **Single DB = one household** — no `user_id` / `tenant_id` on transactions, baselines, etc. |
| Console → API | Vite proxies `/api` → `:8080` (same-origin in dev); no CORS on the API |
| Secrets | Env via `internal/config` (`.env`); LLM/Langfuse keys only |
| Deploy | Local Docker Postgres/Ollama; **no** app Dockerfile / DO app spec yet |

**Implication:** Auth is not a thin middleware sprinkle. It is (a) identity, (b) **data isolation**, (c) CORS/cookies for split app/api hosts, (d) a migration that tags every existing row to the first user/household.

---

## 2. Product rollout (auth methods)

Agreed direction (keep this ladder; do not invent passwords):

| Phase | Who | Sign-in | Notes |
|-------|-----|---------|--------|
| **Internal alpha** | You + invitees | **Google only** | Google OAuth consent screen = Testing; allowlisted emails |
| **Public beta** | Waitlist / open | Google **+ email magic link** | Still no passwords |
| **Later** | Established product | **Passkeys** (and maybe Apple) | After product is sticky |

**Explicit non-goals for a long time**

- User-chosen passwords / password reset flows  
- Social login sprawl (Facebook, etc.)  
- “Login with bank” as IdP  
- Multi-household advisor SaaS (Mandantenverwaltung)

### Why Google first

- Fast for a private alpha (no email deliverability yet).  
- Works on `http://localhost` redirect URIs ([Google web server OAuth](https://developers.google.com/identity/protocols/oauth2/web-server)).  
- Backend confidential client + Authorization Code is the right shape for a Go API.  
- Separates **identity proof** (Google) from **app session** (us).

### Why magic link second (not first)

- Needs reliable transactional email (DO, Postmark, SES, …).  
- Harder to debug locally without mail tooling.  
- Same `auth_identities` model — add provider `email` without schema churn.

### Why no passwords

- Household finance apps that store bank CSVs should not become a password dump.  
- Magic link + Google cover “I have email / I have Google” without credential storage.

---

## 3. User concept (the important product decision)

### 3.1 Two layers: **User** vs **Household**

| Concept | Meaning | Auth? |
|---------|---------|--------|
| **User** | A person who can sign in (Google `sub`, later email) | Yes |
| **Household** | The money world: accounts, transactions, baseline, reviews | Data boundary |

Today’s DB is already a **household**. Production must not pretend every table is “the signed-in user” if you later want a partner on the same books.

**Alpha recommendation (minimal, forward-compatible):**

```text
users 1───1 households          (alpha: one owner = one household)
         └───* memberships      (schema ready; only "owner" used in alpha)
```

- On first Google login: create `user` + `household` + `membership(role=owner)` + migrate/attach existing local data to that household.  
- All money tables get `household_id` (NOT only `user_id`).  
- Session carries `user_id`; request context resolves **active household** (alpha: the only membership).

**Why not `user_id` alone on transactions?**

- Partner / shared household is a common private-finance ask.  
- Re-keying every table later is painful.  
- `user_id` on money rows also confuses “who imported” vs “whose books”.

**Beta+ (optional, not now):** `memberships.role` ∈ `{owner, member}`; invites; still one active household per session cookie.

### 3.2 Identity vs profile

| Store | Fields | Rule |
|-------|--------|------|
| `auth_identities` | `provider`, `provider_subject`, `email`, `email_verified` | **Google `sub` is the join key**, never email |
| `users` | `id`, `display_name?`, `created_at`, … | App person; email is denormalized for UX only |
| `sessions` | opaque id, `user_id`, `expires_at`, `revoked_at?` | Our session, not Google’s |

Email can change at Google; `sub` does not. Linking a second provider later = new `auth_identities` row → same `user_id`.

### 3.3 Local → cloud data story

Pick one for alpha (open decision §10):

| Option | Behavior |
|--------|----------|
| **A. Fresh cloud** | Production DB empty; user imports CSV again |
| **B. Bootstrap first user** | First Google login claims the seeded/migrated household |
| **C. Explicit export/import** | Local tool dumps → cloud import under household |

For a first DigitalOcean alpha with only you: **A or B** is enough. Do not build multi-tenant SaaS yet — build **tenant_id (= household_id) isolation** so a second Google account cannot see the first household’s rows.

---

## 4. Recommended auth architecture

### 4.1 Direction lock (aligned with your sketch)

**React does not talk to Google.** Go owns the full OIDC Authorization Code flow, then issues **our** session cookie. Frontend only:

1. Navigates to `GET {API}/auth/google/start` (full page or `window.location`).  
2. Lands back on `{FRONTEND_URL}/…` after callback.  
3. Calls `GET {API}/api/me` (or `/api/auth/me`) with `credentials: "include"`.

```text
React (app)              Go (api)                    Google
─────────────────────    ─────────────────────────   ─────────────
Click “Mit Google”
  ────────────────────→  /auth/google/start
                           redirect + state/nonce/PKCE
                                                 →  consent
                           /auth/google/callback  ←──
                           validate · upsert user
                           Set-Cookie session
  ←─────────────────────  302 → FRONTEND_URL
fetch /api/me (cookies)
  ────────────────────→  session → user (+ household)
```

Same flow in production; only URLs and `Secure` cookie flags change.

### 4.2 Endpoints (proposed)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/auth/google/start` | Redirect to Google |
| `GET` | `/auth/google/callback` | Code exchange, session, redirect to app |
| `POST` | `/auth/logout` | Revoke session + clear cookie |
| `GET` | `/api/me` | Current user + household summary (auth required) |
| *(later)* | `/auth/magic/start` · `/auth/magic/consume` | Email magic link |

Keep OAuth routes **outside** `/api` or under `/api/auth/*` consistently; document in OpenAPI. Prefer a small `security` scheme: cookie session on all money routes.

### 4.3 Security checklist (must)

Use a maintained library (e.g. `coreos/go-oidc` + `golang.org/x/oauth2`, or equivalent) — do not hand-roll JWT crypto.

- [ ] Authorization **Code** flow (no implicit / no ID token in the browser)  
- [ ] Random **`state`** (CSRF) bound to short-lived server-side store or signed cookie  
- [ ] **`nonce`** in auth request; verify in ID token  
- [ ] **PKCE (S256)** even for confidential server clients ([OAuth BCP / OAuth 2.1 direction](https://datatracker.ietf.org/doc/html/rfc9700))  
- [ ] Validate ID token: signature, `iss`, `aud`, `exp`, `email_verified` (policy: require verified for alpha)  
- [ ] Scopes: **`openid email profile` only**  
- [ ] **Do not persist** Google access/refresh tokens unless you later call Google APIs  
- [ ] Session: cryptographically random opaque id (≥ 256 bits), **hashed at rest** in `sessions`  
- [ ] Idle + absolute expiry (e.g. 14d sliding / 30d absolute) + logout revoke  
- [ ] Separate Google Cloud projects: **Development** vs **Production**

### 4.4 Cookies (important nuance)

Your `__Host-session` idea is **good for production API cookies**, with one clarification:

| Attribute | Production (`api.*`) | Local (`localhost:8080`) |
|-----------|----------------------|---------------------------|
| Name | `__Host-session` | `session` (no `__Host-`: requires `Secure`) |
| `HttpOnly` | yes | yes |
| `Secure` | yes | **false** |
| `SameSite` | `Lax` | `Lax` |
| `Path` | `/` | `/` |
| `Domain` | **omit** (`__Host-` forbids Domain) | omit |

**Why this works with `app.` vs `api.`:** the browser stores the cookie **for the API host**. `fetch('https://api…/api/me', { credentials: 'include' })` from `https://app…` sends it. The SPA never reads the cookie.

**SameSite=Lax is enough** when `app` and `api` are both under the same registrable domain (schemeful same-site). You do **not** need `SameSite=None` for that layout.

**CORS (production):**

- `Access-Control-Allow-Origin: https://app.<domain>` (exact, not `*`)  
- `Access-Control-Allow-Credentials: true`  
- Vite proxy remains optional in local; for **prod-parity local**, use cross-origin `localhost:5173` ↔ `localhost:8080` + CORS like prod.

### 4.5 Middleware contract

After auth ships:

| Route class | Rule |
|-------------|------|
| `/auth/*`, `/health` | Public |
| `/api/*` money + chat + imports | **Require valid session**; set `household_id` on context |
| CLI (`classify`, `migrate`, …) | Unchanged for local ops; cloud ops via SSH/one-off — not browser |

Unauthenticated `/api/*` → `401` with stable error shape. Console shows login gate.

---

## 5. Domains & DigitalOcean (not bought yet)

### 5.1 Hostname plan

Working layout (TLD open — see §10):

| Host | Role |
|------|------|
| `geldlage.<tld>` | Marketing / landing (static) |
| `app.geldlage.<tld>` | Console (SPA) |
| `api.geldlage.<tld>` | Go API + OAuth callback |

Your notes mixed **`.com`** and **`.de`**. Pick one brand domain before creating Google OAuth clients (redirect URIs are exact).

### 5.2 Google Cloud Console

**Project: Geldlage Development**

- OAuth client: Web application  
- Publishing: Testing  
- Redirect: `http://localhost:8080/auth/google/callback`  
- Test users: your Google account(s)

**Project: Geldlage Production**

- Separate client id/secret  
- Redirect: `https://api.geldlage.<tld>/auth/google/callback`  
- Consent screen ready for External + Verification only when leaving Testing

Never put `GOOGLE_CLIENT_SECRET` in `VITE_*`.

### 5.3 Env shape (proposed)

```bash
# shared
APP_ENV=development|production
FRONTEND_URL=http://localhost:5173
CORS_ALLOWED_ORIGINS=http://localhost:5173

GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

SESSION_COOKIE_NAME=session          # production: __Host-session
SESSION_COOKIE_SECURE=false          # production: true
SESSION_TTL_HOURS=336                # example
```

Production swaps URLs to `https://app…` / `https://api…` and `SESSION_COOKIE_SECURE=true`.

### 5.4 DigitalOcean sketch

Reasonable first setup (adjust after domain purchase):

| Piece | DO building block |
|-------|-------------------|
| API | App Platform **service** (Go Dockerfile or buildpack) |
| Console | App Platform **static site** (Vite build) or second service |
| Marketing | Static site on apex |
| Postgres | Managed Database (VPC) — **not** public `0.0.0.0/0` if avoidable |
| Secrets | App Platform encrypted env / DO secrets |

**Cookies on App Platform:** use **custom domains** (not only `*.ondigitalocean.app`) so cookie + CORS behavior matches real hosts. Configure CORS on the API component: exact `https://app…`, credentials allowed ([DO CORS docs](https://docs.digitalocean.com/products/app-platform/how-to/configure-cors-policies/)).

**Alternative (simpler cookies, slightly different DX):** single host `app.geldlage.<tld>` with path routing `/api → backend`, `/ → SPA`. Then same-origin cookies and no CORS. Worth considering if DO ingress path routing is comfortable — **open decision §10**.

---

## 6. Data model (v1)

```text
users
  id              uuid PK
  display_name    text null
  created_at      timestamptz

auth_identities
  id              uuid PK
  user_id         uuid FK → users
  provider        text        -- 'google' | 'email' | …
  provider_subject text       -- Google sub
  email           text
  email_verified  bool
  created_at      timestamptz
  UNIQUE (provider, provider_subject)

households
  id              uuid PK
  name            text        -- default "Mein Haushalt"
  created_at      timestamptz

household_memberships
  household_id    uuid FK
  user_id         uuid FK
  role            text        -- 'owner' | 'member'
  created_at      timestamptz
  PRIMARY KEY (household_id, user_id)

sessions
  id              uuid PK     -- or store only hash
  user_id         uuid FK
  token_hash      bytea       -- SHA-256 of opaque cookie value
  expires_at      timestamptz
  created_at      timestamptz
  revoked_at      timestamptz null
  user_agent      text null
  ip              inet null   -- optional
```

**Money tables (migration):** add `household_id uuid NOT NULL` (after backfill) to at least:

`accounts`, `transactions`, `import_runs`, `financial_baselines`, `money_reviews`, `recurring_series`, `transfer_pairs`, `forecasts`, `decisions`, `actions`, …

System taxonomy (`categories` system rows) can stay global; user-created categories / merchant rules must be household-scoped.

**Every list/query** gains `WHERE household_id = $ctx`. This is the real production blocker, same size as OAuth itself.

---

## 7. Console UX (alpha)

- Unauthenticated: full-page **“Mit Google anmelden”** (marketing can live on apex).  
- No “continue without login” on cloud.  
- Local **dev escape hatch** (optional): `AUTH_DISABLED=true` for single-user offline — **never** in production config. Prefer real Google-on-localhost so alpha matches prod.  
- Sidebar: show email/name from `/api/me` + logout.

---

## 8. Implementation slices (when we build)

Suggested reviewable commits (order matters):

1. **Schema:** `users`, `auth_identities`, `households`, `memberships`, `sessions`  
2. **Tenant column:** `household_id` + backfill strategy + repository/query plumbing  
3. **Session middleware** + `/api/me` + logout (stub login for tests)  
4. **Google OIDC start/callback** + config/env  
5. **Console:** login gate, `credentials: 'include'`, logout, fix API base URL for split hosts  
6. **CORS + cookie flags** for local cross-origin and prod  
7. **Hardening:** revoke, expiry, security tests, OpenAPI `security`  
8. *(Later)* Magic link provider  

Do **not** mix DigitalOcean app spec into the first auth PR unless needed to test cookies on real HTTPS.

---

## 9. What we accept from your sketch vs refine

| Your note | Verdict |
|-----------|---------|
| Backend-owned Google OIDC + own session | **Lock** |
| Google alpha → magic link beta → passkeys later | **Lock** |
| No passwords | **Lock** |
| `users` / `auth_identities` / `sessions` | **Lock** — add **`households` + memberships`** |
| Use Google `sub`, not email, as identity key | **Lock** |
| Separate Google projects for dev/prod | **Lock** |
| `__Host-session` in production | **Lock** (API-host cookie; no `Domain`) |
| `fetch /v1/me` | Prefer **`/api/me`** to match existing `/api` prefix (or migrate all to `/v1` in a dedicated cutover — don’t invent a second versioning style casually) |
| React Google SDK | **Reject** for this architecture |

---

## 10. Decisions (locked 2026-08-09)

| # | Topic | Decision |
|---|--------|----------|
| 1 | Domain TLD | **`.com`** — `geldlage.com` (buy later; develop on localhost) |
| 2 | Host layout | **Split hosts** — `app.geldlage.com` (SPA) + `api.geldlage.com` (Go + OAuth callback). Apex marketing later (out of scope). Rationale: OAuth callback stays on API; cookie scoped to API host (`__Host-session`); clear CORS boundary; matches DigitalOcean multi-component apps. |
| 3 | Local vs prod data | **Prod = empty household on first login.** **Local = existing DB is a seed** claimed once by the first Google user when `AUTH_CLAIM_EXISTING_DATA=true`. |
| 4 | Local auth | **Real Google on localhost** (Testing project). No `AUTH_DISABLED` in normal dev. Tests use session helpers / test doubles, not a prod escape hatch. |
| 5 | Email verified | **Require `email_verified=true`** for Google alpha. |
| 6 | Session lifetime | **14 days sliding**, **30 days absolute**; logout revokes. Opaque cookie; **hash at rest** in `sessions`. |
| 7 | Marketing | **Ignore** for this workstream. |
| 8 | Domain purchase | **End of track** — ship and verify entirely in **dev/localhost** first. |

---

## 11. Risks

| Risk | Mitigation |
|------|------------|
| Ship Google login but forget `household_id` filters | Treat tenancy as same milestone as OAuth; integration tests with two users |
| Cookie not stored on DO starter domains | Custom domain before calling alpha “works” |
| Vite proxy hides CORS bugs | Local mode uses real cross-origin (`VITE_API_BASE_URL`) like prod |
| Pattern/classify CLI on prod DB | Keep CLI ops separate; never expose classify HTTP without auth |
| Scope creep (invites, RBAC, passkeys) | Alpha = Google + 1:1 household only |

---

## 12. Next step

Follow **`plan-auth-implementation.md`** commit-by-commit on localhost. Buy `geldlage.com` and create the Production Google OAuth client only after the local alpha path is green.

---

## Changelog

| Date | Change |
|------|--------|
| 2026-08-09 | Initial research draft from production planning + Google-auth sketch; adds household tenancy and DO/cookie refinements |
| 2026-08-09 | Locked .com, split hosts, empty prod / local seed claim, sessions best practice; pointed to implementation plan |
