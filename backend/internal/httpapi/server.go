package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
	"github.com/PearsSauce/Tqqssl/backend/internal/acmedirectory"
	"github.com/PearsSauce/Tqqssl/backend/internal/acmeregister"
	"github.com/PearsSauce/Tqqssl/backend/internal/auth"
	"github.com/PearsSauce/Tqqssl/backend/internal/config"
	"github.com/PearsSauce/Tqqssl/backend/internal/id"
	"github.com/PearsSauce/Tqqssl/backend/internal/secretbox"
	"github.com/PearsSauce/Tqqssl/backend/internal/store"
)

const sessionCookieName = "tqqssl_personal_session"

type Server struct {
	cfg            config.Config
	store          *store.Store
	secretBox      *secretbox.Box
	acmeAccountKey *acmeaccount.AccountKey
	logger         *slog.Logger
}

type UserDTO struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}

type ACMEStatusDTO struct {
	AccountKeyReady   bool   `json:"accountKeyReady"`
	AccountKeyType    string `json:"accountKeyType,omitempty"`
	DirectoryURL      string `json:"directoryUrl,omitempty"`
	TermsAgreed       bool   `json:"termsAgreed"`
	Ready             bool   `json:"ready"`
	AccountRegistered bool   `json:"accountRegistered"`
	AccountURL        string `json:"accountUrl,omitempty"`
	AccountStatus     string `json:"accountStatus,omitempty"`
	ContactEmail      string `json:"contactEmail,omitempty"`
}

type ACMEDirectoryCheckDTO struct {
	DirectoryURL            string   `json:"directoryUrl"`
	NewNonce                string   `json:"newNonce"`
	NewAccount              string   `json:"newAccount"`
	NewOrder                string   `json:"newOrder"`
	TermsOfService          string   `json:"termsOfService,omitempty"`
	Website                 string   `json:"website,omitempty"`
	ExternalAccountRequired bool     `json:"externalAccountRequired"`
	Warnings                []string `json:"warnings"`
}

type registerACMEAccountRequest struct {
	ContactEmail string `json:"contactEmail"`
}

type ACMEAccountRegistrationDTO struct {
	AccountRegistered bool   `json:"accountRegistered"`
	AccountURL        string `json:"accountUrl"`
	AccountStatus     string `json:"accountStatus"`
	ContactEmail      string `json:"contactEmail"`
}

func New(cfg config.Config, st *store.Store, secretBox *secretbox.Box, acmeAccountKey *acmeaccount.AccountKey, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if secretBox == nil {
		panic("secret box is required")
	}
	return &Server{cfg: cfg, store: st, secretBox: secretBox, acmeAccountKey: acmeAccountKey, logger: logger}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /api/v1/auth/register/options", s.registerOptions)
	mux.HandleFunc("POST /api/v1/auth/register", s.register)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/auth/me", s.me)
	mux.HandleFunc("GET /api/v1/acme/status", s.requireAuth(s.acmeStatus))
	mux.HandleFunc("POST /api/v1/acme/directory/check", s.requireAuth(s.checkACMEDirectory))
	mux.HandleFunc("POST /api/v1/acme/account/register", s.requireAuth(s.registerACMEAccount))
	mux.HandleFunc("GET /api/v1/dns-accounts", s.requireAuth(s.listDNSAccounts))
	mux.HandleFunc("POST /api/v1/dns-accounts", s.requireAuth(s.createDNSAccount))
	mux.HandleFunc("PATCH /api/v1/dns-accounts/{id}", s.requireAuth(s.updateDNSAccount))
	mux.HandleFunc("DELETE /api/v1/dns-accounts/{id}", s.requireAuth(s.deleteDNSAccount))
	mux.HandleFunc("GET /api/v1/certificates/applications", s.requireAuth(s.listCertificateApplications))
	mux.HandleFunc("POST /api/v1/certificates/applications/precheck", s.requireAuth(s.precheckCertificateApplication))
	mux.HandleFunc("POST /api/v1/certificates/applications", s.requireAuth(s.createCertificateApplication))
	mux.HandleFunc("POST /api/v1/certificates/applications/{id}/acme/order", s.requireAuth(s.createCertificateACMEOrder))
	mux.HandleFunc("DELETE /api/v1/certificates/applications/{id}", s.requireAuth(s.deleteCertificateApplication))
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.FrontendOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.cfg.FrontendOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) registerOptions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"allowRegister": s.store.RegisterAvailable()})
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	username := strings.TrimSpace(req.Username)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateUsername(username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		writeError(w, http.StatusBadRequest, "邮箱格式不正确")
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := id.NewUUIDv7()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成用户 ID 失败")
		return
	}
	now := time.Now().UTC()
	user, err := s.store.CreateFirstUser(store.User{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if errors.Is(err, store.ErrRegisterClosed) {
		writeError(w, http.StatusConflict, "个人版只允许初始化一个管理员账号")
		return
	}
	if err != nil {
		s.logger.Error("create user failed", "error", err)
		writeError(w, http.StatusInternalServerError, "注册失败")
		return
	}
	s.issueSession(w, r, user, http.StatusCreated)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	user, err := s.store.FindUserByLogin(req.Username)
	if err != nil || !auth.VerifyPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "账号或密码错误")
		return
	}
	if err := s.store.TouchLastLogin(user.ID, time.Now().UTC()); err != nil {
		s.logger.Warn("touch last login failed", "error", err)
	}
	s.issueSession(w, r, user, http.StatusOK)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.store.DeleteSession(auth.HashSessionToken(cookie.Value))
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	writeJSON(w, http.StatusOK, toUserDTO(user))
}

func (s *Server) acmeStatus(w http.ResponseWriter, _ *http.Request, _ store.User) {
	status := ACMEStatusDTO{
		AccountKeyReady: s.acmeAccountKey != nil,
		DirectoryURL:    s.cfg.ACMEDirectoryURL,
		TermsAgreed:     s.cfg.ACMETermsAgreed,
	}
	if s.acmeAccountKey != nil {
		status.AccountKeyType = s.acmeAccountKey.Type()
	}
	status.Ready = status.AccountKeyReady && status.DirectoryURL != "" && status.TermsAgreed
	if account, err := s.store.GetACMEAccount(); err == nil {
		status.AccountRegistered = account.AccountURL != ""
		status.AccountURL = account.AccountURL
		status.AccountStatus = account.Status
		status.ContactEmail = account.ContactEmail
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) checkACMEDirectory(w http.ResponseWriter, r *http.Request, _ store.User) {
	if strings.TrimSpace(s.cfg.ACMEDirectoryURL) == "" {
		writeError(w, http.StatusBadRequest, "ACME directory URL 未配置")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result, err := acmedirectory.Check(ctx, s.cfg.ACMEDirectoryURL, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ACMEDirectoryCheckDTO{
		DirectoryURL:            result.DirectoryURL,
		NewNonce:                result.NewNonce,
		NewAccount:              result.NewAccount,
		NewOrder:                result.NewOrder,
		TermsOfService:          result.TermsOfService,
		Website:                 result.Website,
		ExternalAccountRequired: result.ExternalAccountRequired,
		Warnings:                result.Warnings,
	})
}

func (s *Server) registerACMEAccount(w http.ResponseWriter, r *http.Request, user store.User) {
	var req registerACMEAccountRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	contactEmail := strings.ToLower(strings.TrimSpace(req.ContactEmail))
	if contactEmail == "" {
		contactEmail = strings.ToLower(strings.TrimSpace(user.Email))
	}
	if _, err := mail.ParseAddress(contactEmail); err != nil {
		writeError(w, http.StatusBadRequest, "ACME 联系邮箱格式不正确")
		return
	}
	if existing, err := s.store.GetACMEAccount(); err == nil && strings.TrimSpace(existing.AccountURL) != "" {
		writeJSON(w, http.StatusOK, toACMEAccountRegistrationDTO(existing))
		return
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.logger.Error("get acme account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "读取 ACME 账号失败")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result, err := acmeregister.Register(ctx, acmeregister.Request{
		DirectoryURL: s.cfg.ACMEDirectoryURL,
		ContactEmail: contactEmail,
		TermsAgreed:  s.cfg.ACMETermsAgreed,
		AccountKey:   s.acmeAccountKey,
	}, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := time.Now().UTC()
	createdAt := now
	if existing, err := s.store.GetACMEAccount(); err == nil && !existing.CreatedAt.IsZero() {
		createdAt = existing.CreatedAt
	}
	account, err := s.store.SaveACMEAccount(store.ACMEAccount{
		DirectoryURL: s.cfg.ACMEDirectoryURL,
		AccountURL:   result.AccountURL,
		ContactEmail: result.ContactEmail,
		Status:       result.Status,
		CreatedAt:    createdAt,
		UpdatedAt:    now,
	})
	if err != nil {
		s.logger.Error("save acme account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "保存 ACME 账号失败")
		return
	}
	writeJSON(w, http.StatusOK, toACMEAccountRegistrationDTO(account))
}

func toACMEAccountRegistrationDTO(account store.ACMEAccount) ACMEAccountRegistrationDTO {
	return ACMEAccountRegistrationDTO{
		AccountRegistered: strings.TrimSpace(account.AccountURL) != "",
		AccountURL:        account.AccountURL,
		AccountStatus:     account.Status,
		ContactEmail:      account.ContactEmail,
	}
}

func (s *Server) currentUser(r *http.Request) (store.User, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return store.User{}, false
	}
	session, err := s.store.GetSession(auth.HashSessionToken(cookie.Value), time.Now().UTC())
	if err != nil {
		return store.User{}, false
	}
	user, err := s.store.GetUser(session.UserID)
	return user, err == nil
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, user store.User, status int) {
	token, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建会话失败")
		return
	}
	now := time.Now().UTC()
	expiresAt := now.Add(s.cfg.SessionTTL)
	if err := s.store.SaveSession(store.Session{TokenHash: tokenHash, UserID: user.ID, CreatedAt: now, ExpiresAt: expiresAt}); err != nil {
		s.logger.Error("save session failed", "error", err)
		writeError(w, http.StatusInternalServerError, "保存会话失败")
		return
	}
	setSessionCookie(w, r, token, expiresAt)
	writeJSON(w, status, map[string]any{"user": toUserDTO(user)})
}

func validateUsername(username string) error {
	if len([]rune(username)) < 3 || len([]rune(username)) > 32 {
		return errors.New("用户名长度需要在 3 到 32 个字符之间")
	}
	for _, r := range username {
		if !(r == '-' || r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return errors.New("用户名只能包含字母、数字、点、下划线和短横线")
		}
	}
	return nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "请求体格式不正确")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"message": message})
}

func toUserDTO(user store.User) UserDTO {
	return UserDTO{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
	}
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func ListenAndServe(ctx context.Context, cfg config.Config, handler http.Handler) error {
	server := &http.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
