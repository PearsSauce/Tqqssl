package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"passwordHash"`
	Role         string     `json:"role"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
}

type Session struct {
	TokenHash string    `json:"tokenHash"`
	UserID    string    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type document struct {
	Users    []User    `json:"users"`
	Sessions []Session `json:"sessions"`
}

type Store struct {
	path string
	mu   sync.Mutex
	doc  document
}

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrRegisterClosed = errors.New("register closed")
)

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.doc = document{Users: []User{}, Sessions: []Session{}}
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		s.doc = document{Users: []User{}, Sessions: []Session{}}
		return nil
	}
	if err := json.Unmarshal(data, &s.doc); err != nil {
		return err
	}
	if s.doc.Users == nil {
		s.doc.Users = []User{}
	}
	if s.doc.Sessions == nil {
		s.doc.Sessions = []Session{}
	}
	return nil
}

func (s *Store) RegisterAvailable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.doc.Users) == 0
}

func (s *Store) CreateFirstUser(user User) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.doc.Users) > 0 {
		return User{}, ErrRegisterClosed
	}
	for _, existing := range s.doc.Users {
		if sameIdentity(existing, user.Username, user.Email) {
			return User{}, ErrAlreadyExists
		}
	}
	s.doc.Users = append(s.doc.Users, user)
	return user, s.saveLocked()
}

func (s *Store) FindUserByLogin(login string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	login = normalize(login)
	for _, user := range s.doc.Users {
		if normalize(user.Username) == login || normalize(user.Email) == login {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) GetUser(id string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.doc.Users {
		if user.ID == id {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *Store) TouchLastLogin(userID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.Users {
		if s.doc.Users[i].ID == userID {
			s.doc.Users[i].LastLoginAt = &now
			s.doc.Users[i].UpdatedAt = now
			return s.saveLocked()
		}
	}
	return ErrNotFound
}

func (s *Store) SaveSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneExpiredLocked(time.Now().UTC())
	s.doc.Sessions = append(s.doc.Sessions, session)
	return s.saveLocked()
}

func (s *Store) GetSession(tokenHash string, now time.Time) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, session := range s.doc.Sessions {
		if session.TokenHash == tokenHash && session.ExpiresAt.After(now) {
			return session, nil
		}
	}
	return Session{}, ErrNotFound
}

func (s *Store) DeleteSession(tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.doc.Sessions[:0]
	for _, session := range s.doc.Sessions {
		if session.TokenHash != tokenHash {
			kept = append(kept, session)
		}
	}
	s.doc.Sessions = kept
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) pruneExpiredLocked(now time.Time) {
	kept := s.doc.Sessions[:0]
	for _, session := range s.doc.Sessions {
		if session.ExpiresAt.After(now) {
			kept = append(kept, session)
		}
	}
	s.doc.Sessions = kept
}

func sameIdentity(user User, username string, email string) bool {
	return normalize(user.Username) == normalize(username) || normalize(user.Email) == normalize(email)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
