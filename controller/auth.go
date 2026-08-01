package controller

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// ParseAllowedEmails builds a lookup set of authorized email addresses from a
// comma-separated list (e.g. from the ALLOWED_EMAILS env var). Entries are
// trimmed and lowercased for case-insensitive comparison.
func ParseAllowedEmails(list string) map[string]bool {
	allowed := make(map[string]bool)
	for _, email := range strings.Split(list, ",") {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
			allowed[email] = true
		}
	}
	return allowed
}

// randString returns a cryptographically secure, URL-safe random string used
// for the OAuth2 state and OIDC nonce values.
func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// setCallbackCookie stores a short-lived, HTTP-only cookie holding the state or
// nonce so the callback can verify the round-trip. Path is "/" so the cookie is
// sent to the callback regardless of which route started the flow, and SameSite
// Lax still allows the cookie on Google's top-level redirect back to us.
// NOTE: cookies are host-scoped — the login flow must be started from the same
// host as OAUTH2_REDIRECT_URL (e.g. don't log in via localhost if the redirect
// URL uses 127.0.0.1), otherwise the callback won't receive these cookies.
func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, c)
}

// cookieValues returns the values of every cookie with the given name. A
// browser can hold several same-named cookies (e.g. leftovers scoped to a
// different Path or Domain), which are all sent on the callback; r.Cookie
// would only return the first one — possibly a stale one shadowing the value
// we just set.
func cookieValues(r *http.Request, name string) []string {
	vals := []string{}
	for _, c := range r.Cookies() {
		if c.Name == name {
			vals = append(vals, c.Value)
		}
	}
	return vals
}

// contains reports whether want is among vals.
func contains(vals []string, want string) bool {
	for _, v := range vals {
		if v == want {
			return true
		}
	}
	return false
}

// HandleLoginPage serves the public login landing page. It is a plain page with
// a "sign in" button and does NOT start the OAuth flow, so landing here after
// logout does not silently re-authenticate the user.
func HandleLoginPage(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("Error rendering login template: %v", err)
		}
	}
}

// HandleLogin starts the OAuth2/OIDC flow: it generates state and nonce values,
// stores them in cookies, and redirects the user to Google's consent screen.
func HandleLogin(oauth2Cnf oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state, err := randString(16)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		nonce, err := randString(16)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		setCallbackCookie(w, r, "state", state)
		setCallbackCookie(w, r, "nonce", nonce)

		http.Redirect(w, r, oauth2Cnf.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
	}
}

// HandleCallback completes the flow:
// - it verifies the state cookie
// - exchanges the authorization code for tokens
// - verifies the ID token and its nonce
// - checks the user's email against the allowlist
// - and starts a session
func HandleCallback(oauth2Cnf oauth2.Config, verifier *oidc.IDTokenVerifier, store *SessionStore, allowedEmails map[string]bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Accept the round-trip if ANY state cookie matches: browsers may send
		// duplicate same-named cookies (see cookieValues), only one of which is
		// the one we just set. CSRF protection is unaffected — an attacker
		// still can't plant a cookie whose value matches the query state.
		states := cookieValues(r, "state")
		if len(states) == 0 {
			cookieNames := []string{}
			for _, c := range r.Cookies() {
				cookieNames = append(cookieNames, c.Name)
			}
			log.Printf("Callback without state cookie: Host=%q, cookies received=%v", r.Host, cookieNames)
			http.Error(w, "state not found", http.StatusBadRequest)
			return
		}
		if !contains(states, r.URL.Query().Get("state")) {
			log.Printf("State mismatch: %d state cookie(s) received, none matches query", len(states))
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := oauth2Cnf.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nonces := cookieValues(r, "nonce")
		if len(nonces) == 0 {
			http.Error(w, "nonce not found", http.StatusBadRequest)
			return
		}
		if !contains(nonces, idToken.Nonce) {
			http.Error(w, "nonce did not match", http.StatusBadRequest)
			return
		}

		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
			Name          string `json:"name"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Only known, Google-verified accounts are allowed in.
		if !claims.EmailVerified || !allowedEmails[strings.ToLower(claims.Email)] {
			log.Printf("Login denied for %q (email_verified=%t)", claims.Email, claims.EmailVerified)
			http.Error(w, "This account is not authorized to access this app.", http.StatusForbidden)
			return
		}

		// Start an authenticated session and hand the browser its session cookie.
		sessionID, err := randString(32)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		if err := store.Create(ctx, sessionID, claims.Email, claims.Name, time.Now().Add(sessionTTL)); err != nil {
			log.Printf("Error persisting session: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    sessionID,
			Path:     "/",
			MaxAge:   int(sessionTTL.Seconds()),
			Secure:   r.TLS != nil,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}
