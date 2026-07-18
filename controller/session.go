package controller

import (
	"net/http"
	"sync"
	"time"
)

// sessionCookieName is the cookie holding the opaque session ID.
const sessionCookieName = "session"

// sessionTTL is how long a session stays valid after login.
const sessionTTL = 24 * time.Hour

// Session holds the authenticated user's identity for the life of the cookie.
type Session struct {
	Email     string
	Name      string
	ExpiresAt time.Time
}

// SessionStore is a concurrency-safe, in-memory store keyed by session ID.
// Sessions are lost on restart; swap this for a persistent store later if needed.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: make(map[string]Session)}
}

func (s *SessionStore) Create(id string, sess Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = sess
}

// Get returns the session for id, or ok=false if it is missing or expired.
func (s *SessionStore) Get(id string) (Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if time.Now().After(sess.ExpiresAt) {
		s.Delete(id)
		return Session{}, false
	}
	return sess, true
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

// RequireAuth wraps a handler so that requests without a valid session are
// redirected to /login.
func RequireAuth(store *SessionStore, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if _, ok := store.Get(c.Value); !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// HandleLogout clears the session both server-side and in the browser.
func HandleLogout(store *SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			store.Delete(c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
