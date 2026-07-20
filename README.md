# Tqqssl 个人版

这是从零编写的管理员自用版本，不从商业版复制业务代码。

## 当前范围

- 本地单管理员注册
- 本地账号/邮箱登录
- HttpOnly 会话 Cookie
- 当前用户查询与退出登录
- React + Vite + HeroUI OSS 前端
- API-only 后端基础骨架

## 明确不包含

- SSO / OIDC
- 多用户、多租户、邀请和兑换
- Agent 与远程部署
- 订阅、支付和商业化额度
- 公告、营销和推广结算
- HeroUI Pro

## 目录

```text
backend/   Go API 与本地数据存储
frontend/  React + Vite + HeroUI OSS
```

## 本地运行

终端一：

```bash
cd backend
go test ./...
go run ./cmd/api
```

终端二：

```bash
cd frontend
corepack pnpm install
corepack pnpm build
corepack pnpm dev
```

默认地址：

- API：`http://localhost:8080`
- 前端：`http://localhost:5173`

用户数据默认保存在 `backend/data/tqqssl-personal.json`，该文件不会提交到 Git。
