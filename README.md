# Tqqssl 个人版

## 项目背景

Tqqssl 个人版面向单管理员自用，目标是提供 DNS 管理和 SSL 证书自动化能力。

当前版本先完成独立运行所需的认证、DNS 账号、ACME 账号注册和证书申请基础闭环，采用：

- 前端静态应用
- Go API 服务
- 本地文件持久化
- 浏览器 HttpOnly 会话

真实 ACME 证书订单、DNS 服务商适配和部署能力将在此基础上按个人版需求逐步实现。

## 已实现功能

- 首个管理员账号初始化注册
- 用户名或邮箱登录
- 登出并清理服务端会话
- 当前登录用户查询
- Argon2id 密码摘要
- UUIDv7 用户 ID
- HttpOnly、SameSite=Lax 会话 Cookie
- `/healthz` 和 `/readyz` 健康检查
- DNS 账号创建、列表和删除
- DNS 账号元数据更新和 SecretKey 轮换
- DNS 账号 SecretKey 使用本地密钥文件加密后写入数据文件，API 响应不返回明文
- ACME 账号私钥自动生成和加载，为后续真实签发做准备
- ACME 就绪状态查询，展示账号私钥、目录 URL 和条款确认状态
- ACME directory 连通性检查，验证核心端点但不创建订单
- ACME 账号注册，默认使用当前管理员邮箱作为 contact，并持久化账号 URL、状态和联系邮箱
- 证书申请记录创建、列表和删除
- 证书申请预检查，不创建记录即可返回规范化域名、DNS 账号和提示
- 证书申请域名基础校验、SAN 去重和小写规范化
- 一个证书申请只允许一种 challenge mode，当前固定为 `dns-01`
- 前端登录、注册、DNS 账号和证书申请控制台
- CORS 配置和本地开发代理

当前版本不包含多用户、SSO/OIDC、Agent、订阅、支付、公告和兑换功能。

## 项目结构

```text
.
├── backend/
│   ├── cmd/api/                  # API 启动入口
│   ├── internal/acmeaccount/      # ACME 账号 P-256 私钥生成和加载
│   ├── internal/acmedirectory/    # ACME directory 连通性和端点检查
│   ├── internal/acmeregister/     # ACME newAccount JWS 注册
│   ├── internal/auth/            # 密码摘要与会话令牌
│   ├── internal/config/          # 环境变量配置
│   ├── internal/httpapi/         # HTTP 路由、处理器和中间件
│   ├── internal/id/              # UUIDv7 生成
│   └── internal/store/           # 本地 JSON 数据存储
└── frontend/
    ├── src/App.tsx               # 页面、认证流程和个人版控制台
    ├── src/api.ts                # API 请求封装
    ├── src/styles.css            # Tailwind CSS 与主题样式
    └── vite.config.ts            # Vite、Tailwind 和开发代理配置
```

## 前端架构

- **框架**：React 19 + TypeScript
- **构建工具**：Vite
- **组件库**：HeroUI OSS v3
- **样式**：Tailwind CSS v4
- **状态管理**：当前使用 React 本地状态
- **路由**：当前使用浏览器 History API，页面范围为登录、注册和控制台
- **认证方式**：通过 `fetch` 携带 HttpOnly Cookie 调用 API，不在 LocalStorage 保存访问令牌
- **控制台能力**：ACME 状态检查和账号注册、DNS 账号管理、证书申请创建和记录列表

前端默认将 `/api` 请求代理到 `http://localhost:8080`。

## 后端架构

- **语言**：Go 1.24+
- **HTTP**：标准库 `net/http`
- **密码存储**：`golang.org/x/crypto/argon2`
- **会话存储**：仅在服务端保存会话令牌摘要
- **数据存储**：JSON 文件，默认路径为 `backend/data/tqqssl-personal.json`，保存用户、会话、DNS 账号、证书申请和 ACME 账号注册状态
- **凭据保护**：DNS SecretKey 使用 AES-GCM 加密，默认密钥文件为 `backend/data/tqqssl-personal.key`
- **ACME 账号**：启动时自动生成或加载 P-256 ECDSA ACME account key，默认路径为 `backend/data/acme-account.key`
- **配置方式**：环境变量
- **接口边界**：所有 DNS 账号和证书申请接口均要求本地管理员会话

> 数据文件、DNS 加密密钥文件和 ACME 账号私钥文件都会以 `0600` 权限写入。密钥文件不应提交到仓库；如果 DNS 加密密钥文件丢失，已保存的 DNS SecretKey 将无法解密。如果 ACME 账号私钥丢失，后续真实签发时需要重新注册 ACME 账号。

### API 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/healthz` | 存活检查 |
| `GET` | `/readyz` | 就绪检查 |
| `GET` | `/api/v1/auth/register/options` | 查询是否允许初始化注册 |
| `POST` | `/api/v1/auth/register` | 创建首个管理员并建立会话 |
| `POST` | `/api/v1/auth/login` | 用户名或邮箱登录 |
| `POST` | `/api/v1/auth/logout` | 删除当前服务端会话 |
| `GET` | `/api/v1/auth/me` | 查询当前登录用户 |
| `GET` | `/api/v1/acme/status` | 查询 ACME 前置配置就绪状态 |
| `POST` | `/api/v1/acme/directory/check` | 检查 ACME directory 连通性和核心端点 |
| `POST` | `/api/v1/acme/account/register` | 注册 ACME 账号并保存账号 URL、状态和联系邮箱 |
| `GET` | `/api/v1/dns-accounts` | 查询 DNS 账号列表 |
| `POST` | `/api/v1/dns-accounts` | 创建 DNS 账号 |
| `PATCH` | `/api/v1/dns-accounts/{id}` | 更新 DNS 账号元数据，可选轮换 SecretKey |
| `DELETE` | `/api/v1/dns-accounts/{id}` | 删除未被证书申请引用的 DNS 账号 |
| `GET` | `/api/v1/certificates/applications` | 查询证书申请记录 |
| `POST` | `/api/v1/certificates/applications/precheck` | 预检查证书申请，不创建记录 |
| `POST` | `/api/v1/certificates/applications` | 创建证书申请记录 |
| `DELETE` | `/api/v1/certificates/applications/{id}` | 删除证书申请记录 |

注册接口只允许在用户数据为空时执行一次。
ACME、DNS 和证书申请接口均要求已登录。

## 如何运行

### 环境要求

- Go 1.24+
- Node.js 20+
- Corepack
- pnpm 11

### 启动后端

```bash
cd backend
go test ./...
go vet ./...
go run ./cmd/api
```

默认监听：

```text
http://localhost:8080
```

### 启动前端

另开终端：

```bash
cd frontend
corepack pnpm install
corepack pnpm typecheck
corepack pnpm build
corepack pnpm dev
```

默认访问：

```text
http://localhost:5173
```

### 配置项

后端支持以下环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TQQSSL_ADDR` | `:8080` | API 监听地址 |
| `TQQSSL_DATA_FILE` | `data/tqqssl-personal.json` | 用户、会话、DNS 账号、证书申请和 ACME 账号状态数据文件 |
| `TQQSSL_SECRET_KEY_FILE` | `data/tqqssl-personal.key` | DNS SecretKey 本地加密密钥文件 |
| `TQQSSL_ACME_ACCOUNT_KEY_FILE` | `data/acme-account.key` | ACME 账号 P-256 私钥文件 |
| `TQQSSL_ACME_DIRECTORY_URL` | 空 | ACME directory URL；为空时不会标记 ACME 就绪 |
| `TQQSSL_ACME_TERMS_AGREED` | `false` | 是否已确认 ACME 服务条款 |
| `TQQSSL_FRONTEND_ORIGIN` | `http://localhost:5173` | CORS 允许的前端来源 |
| `TQQSSL_SESSION_TTL_HOURS` | `24` | 会话有效期，单位为小时 |

示例：

```bash
cd backend
TQQSSL_ADDR=127.0.0.1:8080 \
TQQSSL_FRONTEND_ORIGIN=http://localhost:5173 \
go run ./cmd/api
```

### 重置本地数据

停止后端后删除数据文件，再重新启动。若需要彻底重置 DNS 凭据加密密钥，也同时删除密钥文件：

```bash
rm backend/data/tqqssl-personal.json
rm backend/data/tqqssl-personal.key
rm backend/data/acme-account.key
```

数据文件和密钥文件已加入 Git 忽略规则，不应提交到仓库。
