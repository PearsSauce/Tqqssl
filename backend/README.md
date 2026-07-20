# Tqqssl Personal Backend

从零编写的个人版 API，只包含本地单管理员注册、登录、退出和会话查询基础能力。

不包含：SSO/OIDC、Agent、订阅、支付、公告、兑换、多租户商业逻辑。

## 开发

```bash
go test ./...
go run ./cmd/api
```

默认监听 `:8080`，数据写入 `data/tqqssl-personal.json`。
