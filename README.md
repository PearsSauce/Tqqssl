# Tqqssl 个人版

## 项目背景

Tqqssl 个人版面向单管理员自用，目标是提供 DNS 管理和 SSL 证书自动化能力。

当前版本先完成独立运行所需的认证基础设施，采用：

- 前端静态应用
- Go API 服务
- 本地文件持久化
- 浏览器 HttpOnly 会话

DNS 账号、证书申请和部署能力将在此基础上按个人版需求逐步实现。

## 已实现功能

- 首个管理员账号初始化注册
- 用户名或邮箱登录
- 登出并清理服务端会话
- 当前登录用户查询
- Argon2id 密码摘要
- UUIDv7 用户 ID
- HttpOnly、SameSite=Lax 会话 Cookie
- `/healthz` 和 `/readyz` 健康检查
- 前端登录、注册和基础控制台
- CORS 配置和本地开发代理

当前版本不包含多用户、SSO/OIDC、Agent、订阅、支付、公告和兑换功能。

## 项目结构

```text
.
├── backend/
│   ├── cmd/api/                  # API 启动入口
│   ├── internal/auth/            # 密码摘要与会话令牌
│   ├── internal/config/          # 环境变量配置
│   ├── internal/httpapi/         # HTTP 路由、处理器和中间件
│   ├── internal/id/              # UUIDv7 生成
│   └── internal/store/           # 本地 JSON 数据存储
└── frontend/
    ├── src/App.tsx               # 页面与认证流程
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

前端默认将 `/api` 请求代理到 `http://localhost:8080`。

## 后端架构

- **语言**：Go 1.24+
- **HTTP**：标准库 `net/http`
- **密码存储**：`golang.org/x/crypto/argon2`
- **会话存储**：仅在服务端保存会话令牌摘要
- **数据存储**：JSON 文件，默认路径为 `backend/data/tqqssl-personal.json`
- **配置方式**：环境变量

### 认证接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/healthz` | 存活检查 |
| `GET` | `/readyz` | 就绪检查 |
| `GET` | `/api/v1/auth/register/options` | 查询是否允许初始化注册 |
| `POST` | `/api/v1/auth/register` | 创建首个管理员并建立会话 |
| `POST` | `/api/v1/auth/login` | 用户名或邮箱登录 |
| `POST` | `/api/v1/auth/logout` | 删除当前服务端会话 |
| `GET` | `/api/v1/auth/me` | 查询当前登录用户 |

注册接口只允许在用户数据为空时执行一次。

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
| `TQQSSL_DATA_FILE` | `data/tqqssl-personal.json` | 用户和会话数据文件 |
| `TQQSSL_FRONTEND_ORIGIN` | `http://localhost:5173` | CORS 允许的前端来源 |
| `TQQSSL_SESSION_TTL_HOURS` | `24` | 会话有效期，单位为小时 |

示例：

```bash
cd backend
TQQSSL_ADDR=127.0.0.1:8080 \
TQQSSL_FRONTEND_ORIGIN=http://localhost:5173 \
go run ./cmd/api
```

### 重置本地管理员

停止后端后删除数据文件，再重新启动：

```bash
rm backend/data/tqqssl-personal.json
```

数据文件已加入 Git 忽略规则，不应提交到仓库。
