package main

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const userKey ctxKey = 1

const sessionCookie = "fcc_session"

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func (s *Server) authenticate(username, password string) (*User, bool) {
	var id int64
	var hash string
	if err := s.db.QueryRow(`SELECT id, password_hash FROM users WHERE username=?`, username).Scan(&id, &hash); err != nil {
		bcrypt.CompareHashAndPassword([]byte("$2a$12$invalidinvalidinvalidinvalidinvalidinvalidinvalidinva"), []byte(password)) // temps constant
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, false
	}
	u, err := s.getUserByID(id)
	return u, err == nil
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request, u *User) {
	tok := randomToken("", 32)
	exp := time.Now().Add(14 * 24 * time.Hour)
	s.db.Exec(`INSERT INTO sessions(token_hash, user_id, expires_at) VALUES(?,?,?)`, hashToken(tok), u.ID, exp.UTC().Format(time.RFC3339))
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", Expires: exp})
}

func (s *Server) destroySession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.db.Exec(`DELETE FROM sessions WHERE token_hash=?`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
}

func (s *Server) userFromRequest(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	var uid int64
	var exp string
	if err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE token_hash=?`, hashToken(c.Value)).Scan(&uid, &exp); err != nil {
		return nil
	}
	if t, _ := time.Parse(time.RFC3339, exp); time.Now().After(t) {
		return nil
	}
	u, err := s.getUserByID(uid)
	if err != nil {
		return nil
	}
	return u
}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userKey).(*User)
	return u
}

// requireUser : pages et API du portail. Redirige vers /login pour les pages, 401 pour l'API.
func (s *Server) requireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := s.userFromRequest(r)
		if u == nil {
			if isAPI(r) {
				jsonError(w, http.StatusUnauthorized, "connexion requise")
			} else {
				http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusFound)
			}
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return s.requireUser(func(w http.ResponseWriter, r *http.Request) {
		if !currentUser(r).IsAdmin() {
			if isAPI(r) {
				jsonError(w, http.StatusForbidden, "réservé aux administrateurs")
			} else {
				http.Error(w, "réservé aux administrateurs", http.StatusForbidden)
			}
			return
		}
		next(w, r)
	})
}

func isAPI(r *http.Request) bool { return len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" }

func (s *Server) purgeSessions() {
	s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now())
}
