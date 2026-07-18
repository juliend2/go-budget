package controller

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

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
// nonce so the callback can verify the round-trip.
func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
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

// HandleCallback completes the flow: it verifies the state cookie, exchanges the
// authorization code for tokens, verifies the ID token and its nonce, and
// displays the resulting claims.
func HandleCallback(oauth2Cnf oauth2.Config, verifier *oidc.IDTokenVerifier, store *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		state, err := r.Cookie("state")
		if err != nil {
			http.Error(w, "state not found", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("state") != state.Value {
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

		nonce, err := r.Cookie("nonce")
		if err != nil {
			http.Error(w, "nonce not found", http.StatusBadRequest)
			return
		}
		if idToken.Nonce != nonce.Value {
			http.Error(w, "nonce did not match", http.StatusBadRequest)
			return
		}

		var claims struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := idToken.Claims(&claims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Start an authenticated session and hand the browser its session cookie.
		sessionID, err := randString(32)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		store.Create(sessionID, Session{
			Email:     claims.Email,
			Name:      claims.Name,
			ExpiresAt: time.Now().Add(sessionTTL),
		})
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
