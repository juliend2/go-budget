# Security Audit

Date: 2026-07-21

Findings ordered by severity, with file references and suggested fixes.

## Critical

### 1. No authorization: any Google account can log in — ✅ Fixed (2026-07-21)

`controller/auth.go` (`HandleCallback`)

After verifying the ID token, a session was created for **any** valid Google
account. There was no email allowlist, no `email_verified` check, and no
hosted-domain (`hd`) restriction. Anyone on the internet with a Google account
who found the deployment URL got full read/write access to all financial data.

**Fix applied:** `HandleCallback` now rejects any login whose email is not in
the `ALLOWED_EMAILS` env var (comma-separated, case-insensitive) or whose
`email_verified` claim is false, with HTTP 403. The app refuses to start if
`ALLOWED_EMAILS` is empty, so it cannot run wide open by accident. Denied
attempts are logged server-side.

## High

### 2. Hardcoded weak MongoDB credentials, DB exposed on all interfaces

- `docker-compose.yaml` — `root` / `password` committed to the repo
- `main.go` — same credentials as a hardcoded fallback:
  `mongodb://root:password@localhost:27017/admin`
- `docker-compose.yaml` — `"27017:27017"` publishes MongoDB on `0.0.0.0`, so
  anyone who can reach the host can log in with `root:password` and read/dump
  the whole database.

**Fix:** strong unique password via env var, no hardcoded fallback (fail
startup if `MONGO_URI` is unset), and bind to localhost only:
`"127.0.0.1:27017:27017"`.

## Medium

### 3. `Secure` cookie flag disabled in production

`controller/auth.go` (`setCallbackCookie`, session cookie in `HandleCallback`)

`Secure: r.TLS != nil` — the real deployment is plain HTTP on `:4000` behind a
TLS-terminating proxy (`budget.service` sets
`OAUTH2_REDIRECT_URL=https://...`). Since the Go app never sees TLS, `r.TLS` is
nil and session cookies are set **without** `Secure` in production, making them
sendable over plain HTTP.

**Fix:** set `Secure: true` unconditionally (or honor `X-Forwarded-Proto` via a
proxy-aware middleware), and add HSTS at the proxy.

### 4. No CSRF tokens on state-changing forms

All POST endpoints (`/expense/add`, `/expense/edit`, `/expense/pay`,
`/template/*`) rely solely on `SameSite=Lax`. That blocks most cross-site
POSTs, but it's the only layer, and `/logout` is a GET endpoint
(`views/index.html`) so any site can force-logout users.

**Fix:** add CSRF tokens to forms and make logout POST-only.

### 5. Internal error details leaked to clients

`controller/auth.go`, `controller/handler.go` throughout

`http.Error(w, "Failed to exchange token: "+err.Error(), ...)` and
`fmt.Sprintf("...: %v", err)` expose upstream OAuth errors and MongoDB
internals.

**Fix:** log the detail server-side; return a generic message to the client.

### 6. Missing security headers

No `Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options` /
CSP `frame-ancestors` (clickjacking is possible), or `Referrer-Policy`.

**Fix:** add a small middleware setting these headers on every response.

### 7. `http.ListenAndServe` with no timeouts

`main.go`

The default server has no `ReadHeaderTimeout`/`ReadTimeout`, leaving it open to
Slowloris-style connection exhaustion.

**Fix:** use an `http.Server` with explicit timeouts.

## Low / hardening

- **Session store grows unbounded** — `controller/session.go` only deletes
  expired sessions lazily on `Get`; a background sweeper would cap memory.
- **State/nonce cookies** (`setCallbackCookie`) are never cleared after the
  callback and lack an explicit `Path`/`SameSite`; tighten them and delete
  them in `HandleCallback`.
- **No rate limiting** on `/auth/google/*` endpoints.

## What looks good

- `html/template` auto-escaping everywhere — no XSS found; no unsafe
  `template.HTML` usage.
- No NoSQL injection — all user input goes through
  `primitive.ObjectIDFromHex`/`strconv` before touching `bson` filters.
- OAuth flow correctly uses `crypto/rand` state + nonce, and verifies both.
- Session IDs are 256-bit random, `HttpOnly`, `SameSite=Lax`; logout
  invalidates server-side.
- No secrets committed in git history (the systemd unit uses placeholders).

## Priority

1. ~~Fix finding 1 immediately (open door to all data).~~ Done (2026-07-21).
2. Fix finding 2 (credentials + exposure).
3. Fix finding 3 (cookie `Secure` flag).
