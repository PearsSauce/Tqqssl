package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/PearsSauce/Tqqssl/backend/internal/id"
	"github.com/PearsSauce/Tqqssl/backend/internal/store"
)

const (
	challengeModeDNS01 = "dns-01"
	certificatePending = "pending"
)

type DNSAccountDTO struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	AccessKeyMasked string    `json:"accessKeyMasked,omitempty"`
	HasSecretKey    bool      `json:"hasSecretKey"`
	Remark          string    `json:"remark,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type CertificateApplicationDTO struct {
	ID             string    `json:"id"`
	PrimaryDomain  string    `json:"primaryDomain"`
	SANs           []string  `json:"sans"`
	DNSAccountID   string    `json:"dnsAccountId"`
	DNSAccountName string    `json:"dnsAccountName,omitempty"`
	ChallengeMode  string    `json:"challengeMode"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type createDNSAccountRequest struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
	Remark    string `json:"remark"`
}

type updateDNSAccountRequest struct {
	Name      *string `json:"name"`
	Provider  *string `json:"provider"`
	AccessKey *string `json:"accessKey"`
	SecretKey *string `json:"secretKey"`
	Remark    *string `json:"remark"`
}

type createCertificateApplicationRequest struct {
	PrimaryDomain string   `json:"primaryDomain"`
	SANs          []string `json:"sans"`
	DNSAccountID  string   `json:"dnsAccountId"`
	ChallengeMode string   `json:"challengeMode"`
}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, store.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.currentUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "未登录")
			return
		}
		next(w, r, user)
	}
}

func (s *Server) listDNSAccounts(w http.ResponseWriter, _ *http.Request, _ store.User) {
	accounts := s.store.ListDNSAccounts()
	payload := make([]DNSAccountDTO, 0, len(accounts))
	for _, account := range accounts {
		payload = append(payload, toDNSAccountDTO(account))
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) createDNSAccount(w http.ResponseWriter, r *http.Request, _ store.User) {
	var req createDNSAccountRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	provider := normalizeProvider(req.Provider)
	accessKey := strings.TrimSpace(req.AccessKey)
	secretKey := strings.TrimSpace(req.SecretKey)
	remark := strings.TrimSpace(req.Remark)
	if err := validateDNSAccountMetadata(name, provider, accessKey, remark); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateDNSSecret(secretKey); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	accountID, err := id.NewUUIDv7()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 DNS 账号 ID 失败")
		return
	}
	encryptedSecretKey, err := s.secretBox.Encrypt(secretKey)
	if err != nil {
		s.logger.Error("encrypt dns secret failed", "error", err)
		writeError(w, http.StatusInternalServerError, "加密 DNS 凭据失败")
		return
	}
	now := time.Now().UTC()
	account, err := s.store.CreateDNSAccount(store.DNSAccount{
		ID:        accountID,
		Name:      name,
		Provider:  provider,
		AccessKey: accessKey,
		SecretKey: encryptedSecretKey,
		Remark:    remark,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if errors.Is(err, store.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "DNS 账号名称已存在")
		return
	}
	if err != nil {
		s.logger.Error("create dns account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "创建 DNS 账号失败")
		return
	}
	writeJSON(w, http.StatusCreated, toDNSAccountDTO(account))
}

func (s *Server) updateDNSAccount(w http.ResponseWriter, r *http.Request, _ store.User) {
	accountID := strings.TrimSpace(r.PathValue("id"))
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "DNS 账号 ID 不能为空")
		return
	}
	var req updateDNSAccountRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	existing, err := s.store.GetDNSAccount(accountID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "DNS 账号不存在")
		return
	}
	if err != nil {
		s.logger.Error("get dns account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "读取 DNS 账号失败")
		return
	}

	next := existing
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.Provider != nil {
		next.Provider = normalizeProvider(*req.Provider)
	}
	if req.AccessKey != nil {
		next.AccessKey = strings.TrimSpace(*req.AccessKey)
	}
	if req.Remark != nil {
		next.Remark = strings.TrimSpace(*req.Remark)
	}
	if err := validateDNSAccountMetadata(next.Name, next.Provider, next.AccessKey, next.Remark); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.SecretKey != nil {
		secretKey := strings.TrimSpace(*req.SecretKey)
		if err := validateDNSSecret(secretKey); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		encryptedSecretKey, err := s.secretBox.Encrypt(secretKey)
		if err != nil {
			s.logger.Error("encrypt dns secret failed", "error", err)
			writeError(w, http.StatusInternalServerError, "加密 DNS 凭据失败")
			return
		}
		next.SecretKey = encryptedSecretKey
	}
	next.UpdatedAt = time.Now().UTC()

	account, err := s.store.UpdateDNSAccount(next)
	if errors.Is(err, store.ErrAlreadyExists) {
		writeError(w, http.StatusConflict, "DNS 账号名称已存在")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "DNS 账号不存在")
		return
	}
	if err != nil {
		s.logger.Error("update dns account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "更新 DNS 账号失败")
		return
	}
	writeJSON(w, http.StatusOK, toDNSAccountDTO(account))
}

func (s *Server) deleteDNSAccount(w http.ResponseWriter, r *http.Request, _ store.User) {
	accountID := strings.TrimSpace(r.PathValue("id"))
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "DNS 账号 ID 不能为空")
		return
	}
	err := s.store.DeleteDNSAccount(accountID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "DNS 账号不存在")
	case errors.Is(err, store.ErrInUse):
		writeError(w, http.StatusConflict, "该 DNS 账号已被证书申请使用，不能删除")
	case err != nil:
		s.logger.Error("delete dns account failed", "error", err)
		writeError(w, http.StatusInternalServerError, "删除 DNS 账号失败")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) listCertificateApplications(w http.ResponseWriter, _ *http.Request, _ store.User) {
	accounts := s.store.ListDNSAccounts()
	accountNames := make(map[string]string, len(accounts))
	for _, account := range accounts {
		accountNames[account.ID] = account.Name
	}
	applications := s.store.ListCertificateApplications()
	payload := make([]CertificateApplicationDTO, 0, len(applications))
	for _, application := range applications {
		payload = append(payload, toCertificateApplicationDTO(application, accountNames[application.DNSAccountID]))
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) createCertificateApplication(w http.ResponseWriter, r *http.Request, _ store.User) {
	var req createCertificateApplicationRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	primaryDomain, sans, err := normalizeApplicationDomains(req.PrimaryDomain, req.SANs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dnsAccountID := strings.TrimSpace(req.DNSAccountID)
	if dnsAccountID == "" {
		writeError(w, http.StatusBadRequest, "DNS 账号不能为空")
		return
	}
	challengeMode := strings.ToLower(strings.TrimSpace(req.ChallengeMode))
	if challengeMode == "" {
		challengeMode = challengeModeDNS01
	}
	if challengeMode != challengeModeDNS01 {
		writeError(w, http.StatusBadRequest, "个人版当前仅支持 dns-01 challenge mode")
		return
	}
	applicationID, err := id.NewUUIDv7()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成证书申请 ID 失败")
		return
	}
	now := time.Now().UTC()
	application, err := s.store.CreateCertificateApplication(store.CertificateApplication{
		ID:            applicationID,
		PrimaryDomain: primaryDomain,
		SANs:          sans,
		DNSAccountID:  dnsAccountID,
		ChallengeMode: challengeMode,
		Status:        certificatePending,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "DNS 账号不存在")
		return
	}
	if err != nil {
		s.logger.Error("create certificate application failed", "error", err)
		writeError(w, http.StatusInternalServerError, "创建证书申请失败")
		return
	}
	account, _ := s.store.GetDNSAccount(application.DNSAccountID)
	writeJSON(w, http.StatusCreated, toCertificateApplicationDTO(application, account.Name))
}

func (s *Server) deleteCertificateApplication(w http.ResponseWriter, r *http.Request, _ store.User) {
	applicationID := strings.TrimSpace(r.PathValue("id"))
	if applicationID == "" {
		writeError(w, http.StatusBadRequest, "证书申请 ID 不能为空")
		return
	}
	err := s.store.DeleteCertificateApplication(applicationID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "证书申请不存在")
	case err != nil:
		s.logger.Error("delete certificate application failed", "error", err)
		writeError(w, http.StatusInternalServerError, "删除证书申请失败")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateDNSAccountMetadata(name string, provider string, accessKey string, remark string) error {
	if len([]rune(name)) < 1 || len([]rune(name)) > 64 {
		return errors.New("DNS 账号名称长度需要在 1 到 64 个字符之间")
	}
	if len([]rune(provider)) < 2 || len([]rune(provider)) > 32 {
		return errors.New("DNS 服务商标识长度需要在 2 到 32 个字符之间")
	}
	for _, r := range provider {
		if !(r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z') {
			return errors.New("DNS 服务商标识只能包含小写字母、数字、下划线和短横线")
		}
	}
	if len([]rune(accessKey)) > 256 {
		return errors.New("AccessKey 长度不能超过 256 个字符")
	}
	if len([]rune(remark)) > 300 {
		return errors.New("备注长度不能超过 300 个字符")
	}
	return nil
}

func validateDNSSecret(secretKey string) error {
	if len([]rune(secretKey)) < 1 || len([]rune(secretKey)) > 512 {
		return errors.New("SecretKey 长度需要在 1 到 512 个字符之间")
	}
	return nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func normalizeApplicationDomains(primary string, sans []string) (string, []string, error) {
	primaryDomain, err := normalizeCertificateDomain(primary)
	if err != nil {
		return "", nil, err
	}
	seen := map[string]struct{}{primaryDomain: {}}
	normalizedSANs := make([]string, 0, len(sans))
	for _, value := range sans {
		for _, candidate := range splitDomainInput(value) {
			domain, err := normalizeCertificateDomain(candidate)
			if err != nil {
				return "", nil, err
			}
			if _, ok := seen[domain]; ok {
				continue
			}
			seen[domain] = struct{}{}
			normalizedSANs = append(normalizedSANs, domain)
		}
	}
	if len(seen) > 100 {
		return "", nil, errors.New("一个证书申请最多包含 100 个域名")
	}
	return primaryDomain, normalizedSANs, nil
}

func splitDomainInput(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
}

func normalizeCertificateDomain(value string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(value))
	domain = strings.TrimSuffix(domain, ".")
	if domain == "" {
		return "", errors.New("证书域名不能为空")
	}
	if len(domain) > 253 {
		return "", errors.New("证书域名长度不能超过 253 个字符")
	}
	if strings.HasPrefix(domain, "*.") {
		rest := strings.TrimPrefix(domain, "*.")
		if strings.Contains(rest, "*") {
			return "", errors.New("泛域名通配符只能出现在最左侧")
		}
		if err := validateDNSName(rest); err != nil {
			return "", err
		}
		return "*." + rest, nil
	}
	if strings.Contains(domain, "*") {
		return "", errors.New("泛域名通配符只能出现在最左侧")
	}
	if err := validateDNSName(domain); err != nil {
		return "", err
	}
	return domain, nil
}

func validateDNSName(domain string) error {
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return errors.New("证书域名必须是完整域名")
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return errors.New("证书域名标签长度不正确")
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return errors.New("证书域名标签不能以短横线开头或结尾")
		}
		for _, r := range label {
			if !(r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z') {
				return errors.New("证书域名只能包含字母、数字、点、短横线和最左侧通配符")
			}
		}
	}
	return nil
}

func toDNSAccountDTO(account store.DNSAccount) DNSAccountDTO {
	return DNSAccountDTO{
		ID:              account.ID,
		Name:            account.Name,
		Provider:        account.Provider,
		AccessKeyMasked: maskValue(account.AccessKey),
		HasSecretKey:    account.SecretKey != "",
		Remark:          account.Remark,
		CreatedAt:       account.CreatedAt,
		UpdatedAt:       account.UpdatedAt,
	}
}

func toCertificateApplicationDTO(application store.CertificateApplication, dnsAccountName string) CertificateApplicationDTO {
	return CertificateApplicationDTO{
		ID:             application.ID,
		PrimaryDomain:  application.PrimaryDomain,
		SANs:           append([]string(nil), application.SANs...),
		DNSAccountID:   application.DNSAccountID,
		DNSAccountName: dnsAccountName,
		ChallengeMode:  application.ChallengeMode,
		Status:         application.Status,
		CreatedAt:      application.CreatedAt,
		UpdatedAt:      application.UpdatedAt,
	}
}

func maskValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 8 {
		return "********"
	}
	return string(runes[:4]) + "…" + string(runes[len(runes)-4:])
}
