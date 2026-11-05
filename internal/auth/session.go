package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
)

// Session is a server-side authenticated session. The browser only ever holds
// an opaque random token (HttpOnly cookie); identity lives here.
type Session struct {
	Identity  Identity
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
	IdleUntil time.Time
}

// SessionStore keeps sessions in process memory keyed by the SHA-256 of the
// token, so a memory/log disclosure of the store does not yield usable tokens.
// Sessions do not survive a restart (users sign in again); durable sessions
// are a documented follow-up.
type SessionStore struct {
	TTL     time.Duration // absolute lifetime
	Idle    time.Duration // sliding idle timeout
	Now     func() time.Time
	MaxSess int

	mu sync.Mutex
	m  map[[32]byte]*Session
}

// NewSessionStore constructs a store. Zero values select 12h absolute / 30m idle.
func NewSessionStore(ttl, idle time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	if idle <= 0 {
		idle = 30 * time.Minute
	}
	return &SessionStore{TTL: ttl, Idle: idle, MaxSess: 10000, Now: func() time.Time { return time.Now().UTC() }, m: map[[32]byte]*Session{}}
}

func tokenKey(tok string) [32]byte { return sha256.Sum256([]byte(tok)) }

func randToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Create establishes a session for an authenticated identity.
func (s *SessionStore) Create(_ context.Context, id Identity) (token string, sess Session, err error) {
	token, err = randToken(32)
	if err != nil {
		return "", Session{}, err
	}
	csrf, err := randToken(24)
	if err != nil {
		return "", Session{}, err
	}
	now := s.Now()
	sess = Session{Identity: id, CSRFToken: csrf, CreatedAt: now, ExpiresAt: now.Add(s.TTL), IdleUntil: now.Add(s.Idle)}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepLocked(now)
	if len(s.m) >= s.MaxSess {
		return "", Session{}, fmt.Errorf("%w: session capacity reached", contracts.ErrConflict)
	}
	cp := sess
	s.m[tokenKey(token)] = &cp
	return token, sess, nil
}

// Lookup returns the live session for a token and slides the idle window.
func (s *SessionStore) Lookup(token string) (Session, bool) {
	if token == "" || len(token) > 128 {
		return Session{}, false
	}
	k := tokenKey(token)
	now := s.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[k]
	if !ok {
		return Session{}, false
	}
	if !now.Before(sess.ExpiresAt) || !now.Before(sess.IdleUntil) {
		delete(s.m, k)
		return Session{}, false
	}
	sess.IdleUntil = now.Add(s.Idle)
	if sess.IdleUntil.After(sess.ExpiresAt) {
		sess.IdleUntil = sess.ExpiresAt
	}
	return *sess, true
}

// Delete ends a session.
func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.m, tokenKey(token))
	s.mu.Unlock()
}

func (s *SessionStore) sweepLocked(now time.Time) {
	for k, v := range s.m {
		if !now.Before(v.ExpiresAt) || !now.Before(v.IdleUntil) {
			delete(s.m, k)
		}
	}
}

// CSRFValid compares a presented CSRF token with the session's in constant time.
func CSRFValid(sess Session, presented string) bool {
	return presented != "" && subtle.ConstantTimeCompare([]byte(sess.CSRFToken), []byte(presented)) == 1
}
