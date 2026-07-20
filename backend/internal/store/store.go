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

type DNSAccount struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	AccessKey string    `json:"accessKey"`
	SecretKey string    `json:"secretKey"`
	Remark    string    `json:"remark,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CertificateApplication struct {
	ID                string    `json:"id"`
	PrimaryDomain     string    `json:"primaryDomain"`
	SANs              []string  `json:"sans"`
	DNSAccountID      string    `json:"dnsAccountId"`
	ChallengeMode     string    `json:"challengeMode"`
	Status            string    `json:"status"`
	OrderURL          string    `json:"orderUrl,omitempty"`
	OrderStatus       string    `json:"orderStatus,omitempty"`
	AuthorizationURLs []string  `json:"authorizationUrls,omitempty"`
	FinalizeURL       string    `json:"finalizeUrl,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type CertificateOrder struct {
	OrderURL          string
	OrderStatus       string
	AuthorizationURLs []string
	FinalizeURL       string
}

type ACMEAccount struct {
	DirectoryURL string    `json:"directoryUrl"`
	AccountURL   string    `json:"accountUrl"`
	ContactEmail string    `json:"contactEmail"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type document struct {
	Users                   []User                   `json:"users"`
	Sessions                []Session                `json:"sessions"`
	DNSAccounts             []DNSAccount             `json:"dnsAccounts"`
	CertificateApplications []CertificateApplication `json:"certificateApplications"`
	ACMEAccount             *ACMEAccount             `json:"acmeAccount,omitempty"`
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
	ErrInUse          = errors.New("in use")
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
		s.doc = newDocument()
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		s.doc = newDocument()
		return nil
	}
	if err := json.Unmarshal(data, &s.doc); err != nil {
		return err
	}
	s.ensureDocument()
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

func (s *Store) ListDNSAccounts() []DNSAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts := make([]DNSAccount, 0, len(s.doc.DNSAccounts))
	for _, account := range s.doc.DNSAccounts {
		accounts = append(accounts, account)
	}
	return accounts
}

func (s *Store) GetDNSAccount(id string) (DNSAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, account := range s.doc.DNSAccounts {
		if account.ID == id {
			return account, nil
		}
	}
	return DNSAccount{}, ErrNotFound
}

func (s *Store) CreateDNSAccount(account DNSAccount) (DNSAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.doc.DNSAccounts {
		if existing.ID == account.ID || normalize(existing.Name) == normalize(account.Name) {
			return DNSAccount{}, ErrAlreadyExists
		}
	}
	s.doc.DNSAccounts = append(s.doc.DNSAccounts, account)
	return account, s.saveLocked()
}

func (s *Store) UpdateDNSAccount(account DNSAccount) (DNSAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, existing := range s.doc.DNSAccounts {
		if existing.ID == account.ID {
			index = i
			continue
		}
		if normalize(existing.Name) == normalize(account.Name) {
			return DNSAccount{}, ErrAlreadyExists
		}
	}
	if index == -1 {
		return DNSAccount{}, ErrNotFound
	}
	s.doc.DNSAccounts[index] = account
	return account, s.saveLocked()
}

func (s *Store) DeleteDNSAccount(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, application := range s.doc.CertificateApplications {
		if application.DNSAccountID == id {
			return ErrInUse
		}
	}
	kept := s.doc.DNSAccounts[:0]
	found := false
	for _, account := range s.doc.DNSAccounts {
		if account.ID == id {
			found = true
			continue
		}
		kept = append(kept, account)
	}
	if !found {
		return ErrNotFound
	}
	s.doc.DNSAccounts = kept
	return s.saveLocked()
}

func (s *Store) EncryptPlaintextDNSSecrets(encrypt func(string) (string, error), isEncrypted func(string) bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	migrated := 0
	for i := range s.doc.DNSAccounts {
		secretKey := strings.TrimSpace(s.doc.DNSAccounts[i].SecretKey)
		if secretKey == "" || isEncrypted(secretKey) {
			continue
		}
		encrypted, err := encrypt(secretKey)
		if err != nil {
			return migrated, err
		}
		s.doc.DNSAccounts[i].SecretKey = encrypted
		s.doc.DNSAccounts[i].UpdatedAt = time.Now().UTC()
		migrated++
	}
	if migrated == 0 {
		return 0, nil
	}
	return migrated, s.saveLocked()
}

func (s *Store) ListCertificateApplications() []CertificateApplication {
	s.mu.Lock()
	defer s.mu.Unlock()
	applications := make([]CertificateApplication, 0, len(s.doc.CertificateApplications))
	for _, application := range s.doc.CertificateApplications {
		application.SANs = append([]string(nil), application.SANs...)
		application.AuthorizationURLs = append([]string(nil), application.AuthorizationURLs...)
		applications = append(applications, application)
	}
	return applications
}

func (s *Store) GetCertificateApplication(id string) (CertificateApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, application := range s.doc.CertificateApplications {
		if application.ID == id {
			application.SANs = append([]string(nil), application.SANs...)
			application.AuthorizationURLs = append([]string(nil), application.AuthorizationURLs...)
			return application, nil
		}
	}
	return CertificateApplication{}, ErrNotFound
}

func (s *Store) CreateCertificateApplication(application CertificateApplication) (CertificateApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dnsAccountExists := false
	for _, account := range s.doc.DNSAccounts {
		if account.ID == application.DNSAccountID {
			dnsAccountExists = true
			break
		}
	}
	if !dnsAccountExists {
		return CertificateApplication{}, ErrNotFound
	}
	for _, existing := range s.doc.CertificateApplications {
		if existing.ID == application.ID {
			return CertificateApplication{}, ErrAlreadyExists
		}
	}
	application.SANs = append([]string(nil), application.SANs...)
	application.AuthorizationURLs = append([]string(nil), application.AuthorizationURLs...)
	s.doc.CertificateApplications = append(s.doc.CertificateApplications, application)
	return application, s.saveLocked()
}

func (s *Store) SaveCertificateApplicationOrder(id string, applicationStatus string, order CertificateOrder, now time.Time) (CertificateApplication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.doc.CertificateApplications {
		if s.doc.CertificateApplications[i].ID == id {
			s.doc.CertificateApplications[i].Status = applicationStatus
			s.doc.CertificateApplications[i].OrderURL = order.OrderURL
			s.doc.CertificateApplications[i].OrderStatus = order.OrderStatus
			s.doc.CertificateApplications[i].AuthorizationURLs = append([]string(nil), order.AuthorizationURLs...)
			s.doc.CertificateApplications[i].FinalizeURL = order.FinalizeURL
			s.doc.CertificateApplications[i].UpdatedAt = now
			application := s.doc.CertificateApplications[i]
			application.SANs = append([]string(nil), application.SANs...)
			application.AuthorizationURLs = append([]string(nil), application.AuthorizationURLs...)
			return application, s.saveLocked()
		}
	}
	return CertificateApplication{}, ErrNotFound
}

func (s *Store) DeleteCertificateApplication(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.doc.CertificateApplications[:0]
	found := false
	for _, application := range s.doc.CertificateApplications {
		if application.ID == id {
			found = true
			continue
		}
		kept = append(kept, application)
	}
	if !found {
		return ErrNotFound
	}
	s.doc.CertificateApplications = kept
	return s.saveLocked()
}

func (s *Store) GetACMEAccount() (ACMEAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doc.ACMEAccount == nil {
		return ACMEAccount{}, ErrNotFound
	}
	return *s.doc.ACMEAccount, nil
}

func (s *Store) SaveACMEAccount(account ACMEAccount) (ACMEAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.doc.ACMEAccount = &account
	return account, s.saveLocked()
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

func newDocument() document {
	return document{
		Users:                   []User{},
		Sessions:                []Session{},
		DNSAccounts:             []DNSAccount{},
		CertificateApplications: []CertificateApplication{},
	}
}

func (s *Store) ensureDocument() {
	if s.doc.Users == nil {
		s.doc.Users = []User{}
	}
	if s.doc.Sessions == nil {
		s.doc.Sessions = []Session{}
	}
	if s.doc.DNSAccounts == nil {
		s.doc.DNSAccounts = []DNSAccount{}
	}
	if s.doc.CertificateApplications == nil {
		s.doc.CertificateApplications = []CertificateApplication{}
	}
}

func sameIdentity(user User, username string, email string) bool {
	return normalize(user.Username) == normalize(username) || normalize(user.Email) == normalize(email)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
