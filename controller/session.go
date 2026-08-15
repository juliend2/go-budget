package controller

import (
	"context"
	"log"
	"net/http"
	"time"

	"desrosiers.org/budget/model"
)

// sessionCookieName is the cookie holding the opaque session ID.
const sessionCookieName = "session"

// sessionTTL is how long a session stays valid after login.
const sessionTTL = 24 * time.Hour * 7 // how many days

// SessionRepository is the persistence the session store needs. It is satisfied
// by *repository.MongoDBRepository.
type SessionRepository interface {
	CreateSession(ctx context.Context, sess *model.Session) error
	GetSessionBySecureID(ctx context.Context, secureID string) (*model.Session, error)
	DeleteSession(ctx context.Context, secureID string) error
}

// SessionStore persists authenticated sessions in MongoDB, keyed by the opaque
// secure ID stored in the browser cookie. Sessions now survive restarts.
type SessionStore struct {
	repo SessionRepository
}

func NewSessionStore(repo SessionRepository) *SessionStore {
	return &SessionStore{repo: repo}
}

// Create persists a new session for secureID.
func (s *SessionStore) Create(ctx context.Context, secureID, email, name string, expiresAt time.Time) error {
	return s.repo.CreateSession(ctx, &model.Session{
		SecureID:  secureID,
		Email:     email,
		Name:      name,
		ExpiresAt: expiresAt,
	})
}

// Get returns the session for secureID, or ok=false if it is missing or expired.
// Expired sessions are deleted as a side effect.
func (s *SessionStore) Get(ctx context.Context, secureID string) (*model.Session, bool) {
	sess, err := s.repo.GetSessionBySecureID(ctx, secureID)
	if err != nil {
		log.Printf("Error fetching session: %v", err)
		return nil, false
	}
	if sess == nil {
		return nil, false
	}
	if time.Now().After(sess.ExpiresAt) {
		if err := s.repo.DeleteSession(ctx, secureID); err != nil {
			log.Printf("Error deleting expired session: %v", err)
		}
		return nil, false
	}
	return sess, true
}

func (s *SessionStore) Delete(ctx context.Context, secureID string) error {
	return s.repo.DeleteSession(ctx, secureID)
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
		if _, ok := store.Get(r.Context(), c.Value); !ok {
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
			if err := store.Delete(r.Context(), c.Value); err != nil {
				log.Printf("Error deleting session on logout: %v", err)
			}
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
