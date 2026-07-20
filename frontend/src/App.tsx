import { Alert, Button, Card, Description, FieldError, Form, Input, Label, Spinner, TextArea, TextField } from "@heroui/react";
import { useEffect, useMemo, useState } from "react";

import { ApiError, apiRequest } from "./api";
import type { CertificateApplication, DNSAccount, RegisterOptions, User } from "./types";

type AuthResponse = {
  user: User;
};

type Route = "/" | "/login" | "/register";

const routeSet = new Set<Route>(["/", "/login", "/register"]);

export default function App() {
  const [route, setRoute] = useState<Route>(() => normalizeRoute(window.location.pathname));
  const [user, setUser] = useState<User | null>(null);
  const [options, setOptions] = useState<RegisterOptions | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    function handlePopState() {
      setRoute(normalizeRoute(window.location.pathname));
    }
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function bootstrap() {
      const [meResult, optionsResult] = await Promise.allSettled([
        apiRequest<User>("/auth/me"),
        apiRequest<RegisterOptions>("/auth/register/options")
      ]);
      if (cancelled) {
        return;
      }
      if (meResult.status === "fulfilled") {
        setUser(meResult.value);
      }
      if (optionsResult.status === "fulfilled") {
        setOptions(optionsResult.value);
      } else {
        setOptions({ allowRegister: false });
      }
      setLoading(false);
    }
    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, []);

  const navigate = (nextRoute: Route) => {
    window.history.pushState({}, "", nextRoute);
    setRoute(nextRoute);
  };

  const handleAuthSuccess = (nextUser: User) => {
    setUser(nextUser);
    setOptions({ allowRegister: false });
    navigate("/");
  };

  const logout = async () => {
    await apiRequest<void>("/auth/logout", { method: "POST" });
    setUser(null);
    navigate("/login");
  };

  if (loading) {
    return <LoadingScreen />;
  }

  if (user) {
    return <Dashboard user={user} onLogout={logout} />;
  }

  if (route === "/login" || !options?.allowRegister) {
    return <LoginPage allowRegister={Boolean(options?.allowRegister)} onNavigate={navigate} onSuccess={handleAuthSuccess} />;
  }

  return <RegisterPage onNavigate={navigate} onSuccess={handleAuthSuccess} />;
}

function LoadingScreen() {
  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <Card className="w-full max-w-sm items-center gap-4 p-8 text-center">
        <Spinner size="lg" />
        <Card.Header>
          <Card.Title>正在载入个人版控制台</Card.Title>
          <Card.Description>检查本地会话与初始化状态。</Card.Description>
        </Card.Header>
      </Card>
    </main>
  );
}

function LoginPage({ allowRegister, onNavigate, onSuccess }: {
  allowRegister: boolean;
  onNavigate: (route: Route) => void;
  onSuccess: (user: User) => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setPending(true);
    setError("");
    try {
      const result = await apiRequest<AuthResponse>("/auth/login", {
        method: "POST",
        body: { username, password }
      });
      onSuccess(result.user);
    } catch (err) {
      setError(errorMessage(err, "登录失败"));
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthShell title="登录个人版" description="使用本地管理员账号进入控制台。">
      <Form
        className="grid gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        {error ? <InlineAlert status="danger" title={error} /> : null}
        <TextField isRequired fullWidth name="username" value={username} onChange={setUsername}>
          <Label>邮箱或用户名</Label>
          <Input autoComplete="username" placeholder="admin@example.com" />
        </TextField>
        <TextField isRequired fullWidth name="password" type="password" value={password} onChange={setPassword}>
          <Label>密码</Label>
          <Input autoComplete="current-password" placeholder="输入密码" />
        </TextField>
        <Button fullWidth isPending={pending} type="submit">
          {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}登录</>}
        </Button>
      </Form>
      {allowRegister ? (
        <Button fullWidth variant="ghost" onPress={() => onNavigate("/register")}>还没有账号，初始化管理员</Button>
      ) : null}
    </AuthShell>
  );
}

function RegisterPage({ onNavigate, onSuccess }: {
  onNavigate: (route: Route) => void;
  onSuccess: (user: User) => void;
}) {
  const [username, setUsername] = useState("admin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const passwordInvalid = password.length > 0 && password.length < 10;

  async function submit() {
    setPending(true);
    setError("");
    try {
      const result = await apiRequest<AuthResponse>("/auth/register", {
        method: "POST",
        body: { username, email, password }
      });
      onSuccess(result.user);
    } catch (err) {
      setError(errorMessage(err, "注册失败"));
    } finally {
      setPending(false);
    }
  }

  return (
    <AuthShell title="初始化管理员" description="个人版只允许一个本地管理员账号。">
      <Form
        className="grid gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        {error ? <InlineAlert status="danger" title={error} /> : null}
        <TextField isRequired fullWidth name="username" value={username} onChange={setUsername}>
          <Label>用户名</Label>
          <Input autoComplete="username" placeholder="admin" />
        </TextField>
        <TextField isRequired fullWidth name="email" type="email" value={email} onChange={setEmail}>
          <Label>邮箱</Label>
          <Input autoComplete="email" placeholder="admin@example.com" />
        </TextField>
        <TextField
          isRequired
          fullWidth
          isInvalid={passwordInvalid}
          name="password"
          type="password"
          value={password}
          onChange={setPassword}
        >
          <Label>密码</Label>
          <Input autoComplete="new-password" placeholder="至少 10 位，包含三类字符" />
          {passwordInvalid ? <FieldError>密码至少需要 10 个字符。</FieldError> : <Description>建议包含大小写、数字和符号。</Description>}
        </TextField>
        <Button fullWidth isPending={pending} type="submit">
          {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}创建管理员</>}
        </Button>
      </Form>
      <Button fullWidth variant="ghost" onPress={() => onNavigate("/login")}>已有账号，去登录</Button>
    </AuthShell>
  );
}

function AuthShell({ title, description, children }: { title: string; description: string; children: React.ReactNode }) {
  return (
    <main className="grid min-h-screen place-items-center px-5 py-10">
      <section className="grid w-full max-w-5xl gap-8 lg:grid-cols-[1.05fr_0.95fr]">
        <div className="flex flex-col justify-center gap-8">
          <div>
            <div className="mb-5 flex size-12 items-center justify-center rounded-2xl bg-slate-950 text-lg font-semibold text-white shadow-xl shadow-slate-900/10">T</div>
            <h1 className="max-w-xl text-4xl font-semibold tracking-tight text-slate-950 sm:text-5xl">Tqqssl 个人版</h1>
            <p className="mt-5 max-w-xl text-base leading-7 text-slate-600">只保留管理员自用的证书与 DNS 自动化基础能力。本版本从零实现，不包含商业化后端、SSO、Agent、订阅、支付、公告或兑换模块。</p>
          </div>
          <div className="grid gap-3 text-sm text-slate-600 sm:grid-cols-3">
            <Feature title="本地账号" description="单管理员注册登录" />
            <Feature title="HttpOnly 会话" description="浏览器不保存令牌" />
            <Feature title="API 优先" description="后续只接 API 部署" />
          </div>
        </div>
        <Card className="w-full gap-6 p-6 shadow-2xl shadow-blue-950/10 sm:p-8">
          <Card.Header>
            <Card.Title className="text-2xl">{title}</Card.Title>
            <Card.Description>{description}</Card.Description>
          </Card.Header>
          <Card.Content className="grid gap-4">{children}</Card.Content>
        </Card>
      </section>
    </main>
  );
}

function Feature({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-3xl border border-white/70 bg-white/70 p-4 shadow-sm backdrop-blur">
      <div className="font-medium text-slate-950">{title}</div>
      <div className="mt-1 text-xs leading-5 text-slate-500">{description}</div>
    </div>
  );
}

function Dashboard({ user, onLogout }: { user: User; onLogout: () => Promise<void> }) {
  const createdAt = useMemo(() => formatDateTime(user.createdAt), [user.createdAt]);
  const [pending, setPending] = useState(false);
  const [dnsAccounts, setDNSAccounts] = useState<DNSAccount[]>([]);
  const [applications, setApplications] = useState<CertificateApplication[]>([]);
  const [resourcesLoading, setResourcesLoading] = useState(true);
  const [resourcesError, setResourcesError] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function loadResources() {
      setResourcesLoading(true);
      setResourcesError("");
      try {
        const [nextDNSAccounts, nextApplications] = await Promise.all([
          apiRequest<DNSAccount[]>("/dns-accounts"),
          apiRequest<CertificateApplication[]>("/certificates/applications")
        ]);
        if (!cancelled) {
          setDNSAccounts(nextDNSAccounts);
          setApplications(nextApplications);
        }
      } catch (err) {
        if (!cancelled) {
          setResourcesError(errorMessage(err, "加载 DNS 账号和证书申请失败"));
        }
      } finally {
        if (!cancelled) {
          setResourcesLoading(false);
        }
      }
    }
    void loadResources();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <main className="min-h-screen px-5 py-8">
      <section className="mx-auto grid max-w-6xl gap-6">
        <header className="flex flex-col gap-4 rounded-[2rem] border border-white/70 bg-white/80 p-6 shadow-xl shadow-blue-950/5 backdrop-blur md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-sm text-slate-500">个人版控制台</p>
            <h1 className="mt-1 text-3xl font-semibold tracking-tight text-slate-950">欢迎，{user.username}</h1>
            <p className="mt-2 text-sm text-slate-500">账号创建于 {createdAt}</p>
          </div>
          <Button
            variant="secondary"
            isPending={pending}
            onPress={() => {
              setPending(true);
              void onLogout().finally(() => setPending(false));
            }}
          >
            退出登录
          </Button>
        </header>

        <div className="grid gap-4 md:grid-cols-3">
          <SummaryCard title="DNS 账号" value={`${dnsAccounts.length} 个`} description="本地保存 DNS API 凭据，接口响应不返回 SecretKey。" />
          <SummaryCard title="证书申请" value={`${applications.length} 条`} description="当前建立申请记录，challenge mode 固定为 dns-01。" />
          <SummaryCard title="商业模块" value="未启用" description="没有多用户、SSO、Agent、订阅、支付、公告和兑换。" />
        </div>

        {resourcesError ? <InlineAlert status="danger" title={resourcesError} /> : null}
        {resourcesLoading ? (
          <Card className="items-center gap-3 p-6 text-center">
            <Spinner />
            <Card.Description>正在加载 DNS 账号与证书申请记录。</Card.Description>
          </Card>
        ) : (
          <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
            <DNSAccountsPanel
              accounts={dnsAccounts}
              onCreated={(account) => setDNSAccounts((current) => [account, ...current])}
              onDeleted={(accountID) => setDNSAccounts((current) => current.filter((account) => account.id !== accountID))}
            />
            <CertificateApplicationsPanel
              applications={applications}
              dnsAccounts={dnsAccounts}
              onCreated={(application) => setApplications((current) => [application, ...current])}
            />
          </div>
        )}

        <Card className="p-6">
          <Card.Header>
            <Card.Title>实现边界</Card.Title>
            <Card.Description>个人版当前完成 DNS 账号和证书申请基础闭环，后续再接入真实 ACME 签发与 DNS 提供商适配。</Card.Description>
          </Card.Header>
          <Card.Content>
            <InlineAlert status="accent" title="当前是干净个人版实现" description="本仓库没有引入商业化后端、SSO/OIDC、Agent、订阅、支付、公告或兑换模块。" />
          </Card.Content>
        </Card>
      </section>
    </main>
  );
}

function DNSAccountsPanel({ accounts, onCreated, onDeleted }: {
  accounts: DNSAccount[];
  onCreated: (account: DNSAccount) => void;
  onDeleted: (accountID: string) => void;
}) {
  const [name, setName] = useState("");
  const [provider, setProvider] = useState("alidns");
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [remark, setRemark] = useState("");
  const [pending, setPending] = useState(false);
  const [deletingID, setDeletingID] = useState("");
  const [error, setError] = useState("");

  async function submit() {
    setPending(true);
    setError("");
    try {
      const account = await apiRequest<DNSAccount>("/dns-accounts", {
        method: "POST",
        body: { name, provider, accessKey, secretKey, remark }
      });
      onCreated(account);
      setName("");
      setAccessKey("");
      setSecretKey("");
      setRemark("");
    } catch (err) {
      setError(errorMessage(err, "创建 DNS 账号失败"));
    } finally {
      setPending(false);
    }
  }

  async function deleteAccount(accountID: string) {
    setDeletingID(accountID);
    setError("");
    try {
      await apiRequest<void>(`/dns-accounts/${accountID}`, { method: "DELETE" });
      onDeleted(accountID);
    } catch (err) {
      setError(errorMessage(err, "删除 DNS 账号失败"));
    } finally {
      setDeletingID("");
    }
  }

  return (
    <Card className="gap-6 p-6">
      <Card.Header>
        <Card.Title>DNS 账号</Card.Title>
        <Card.Description>保存个人签发证书所需的 DNS API 凭据。SecretKey 仅写入本地数据文件，不会从 API 返回。</Card.Description>
      </Card.Header>
      <Card.Content className="grid gap-6">
        <Form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          {error ? <InlineAlert status="danger" title={error} /> : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <TextField isRequired fullWidth name="dnsName" value={name} onChange={setName}>
              <Label>账号名称</Label>
              <Input placeholder="阿里云主账号" />
            </TextField>
            <TextField isRequired fullWidth name="provider" value={provider} onChange={setProvider}>
              <Label>服务商标识</Label>
              <Input placeholder="alidns / dnspod / cloudflare" />
              <Description>仅使用小写字母、数字、下划线和短横线。</Description>
            </TextField>
          </div>
          <TextField fullWidth name="accessKey" value={accessKey} onChange={setAccessKey}>
            <Label>AccessKey</Label>
            <Input autoComplete="off" placeholder="可选，按服务商要求填写" />
          </TextField>
          <TextField isRequired fullWidth name="secretKey" type="password" value={secretKey} onChange={setSecretKey}>
            <Label>SecretKey / API Token</Label>
            <Input autoComplete="new-password" placeholder="创建后不会再展示明文" />
          </TextField>
          <div className="grid gap-2">
            <Label htmlFor="dns-remark">备注</Label>
            <TextArea
              fullWidth
              id="dns-remark"
              maxLength={300}
              placeholder="例如：只用于 example.com 的 DNS-01 验证"
              rows={3}
              value={remark}
              onChange={(event) => setRemark(event.target.value)}
            />
          </div>
          <Button isPending={pending} type="submit">
            {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}保存 DNS 账号</>}
          </Button>
        </Form>

        <div className="grid gap-3">
          {accounts.length === 0 ? (
            <EmptyState title="还没有 DNS 账号" description="先新增 DNS 账号，再创建证书申请。" />
          ) : (
            accounts.map((account) => (
              <div key={account.id} className="rounded-3xl border border-slate-200/80 bg-slate-50/80 p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <div className="font-medium text-slate-950">{account.name}</div>
                    <div className="mt-1 text-sm text-slate-500">{account.provider} · {account.accessKeyMasked || "未填写 AccessKey"}</div>
                    {account.remark ? <div className="mt-2 text-sm leading-6 text-slate-500">{account.remark}</div> : null}
                  </div>
                  <Button
                    variant="secondary"
                    isPending={deletingID === account.id}
                    onPress={() => void deleteAccount(account.id)}
                  >
                    删除
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>
      </Card.Content>
    </Card>
  );
}

function CertificateApplicationsPanel({ applications, dnsAccounts, onCreated }: {
  applications: CertificateApplication[];
  dnsAccounts: DNSAccount[];
  onCreated: (application: CertificateApplication) => void;
}) {
  const [primaryDomain, setPrimaryDomain] = useState("");
  const [sansText, setSANsText] = useState("");
  const [selectedDNSAccountID, setSelectedDNSAccountID] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (dnsAccounts.length === 0) {
      setSelectedDNSAccountID("");
      return;
    }
    if (!dnsAccounts.some((account) => account.id === selectedDNSAccountID)) {
      setSelectedDNSAccountID(dnsAccounts[0].id);
    }
  }, [dnsAccounts, selectedDNSAccountID]);

  async function submit() {
    setPending(true);
    setError("");
    try {
      const application = await apiRequest<CertificateApplication>("/certificates/applications", {
        method: "POST",
        body: {
          primaryDomain,
          sans: parseDomainList(sansText),
          dnsAccountId: selectedDNSAccountID,
          challengeMode: "dns-01"
        }
      });
      onCreated(application);
      setPrimaryDomain("");
      setSANsText("");
    } catch (err) {
      setError(errorMessage(err, "创建证书申请失败"));
    } finally {
      setPending(false);
    }
  }

  return (
    <Card className="gap-6 p-6">
      <Card.Header>
        <Card.Title>证书申请</Card.Title>
        <Card.Description>一个申请只允许一种 challenge mode；个人版当前固定为 DNS-01，并引用一个 DNS 账号。</Card.Description>
      </Card.Header>
      <Card.Content className="grid gap-6">
        <Form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          {error ? <InlineAlert status="danger" title={error} /> : null}
          {dnsAccounts.length === 0 ? <InlineAlert status="accent" title="需要先创建 DNS 账号" description="证书申请会校验 DNS 账号是否存在。" /> : null}
          <TextField isRequired fullWidth name="primaryDomain" value={primaryDomain} onChange={setPrimaryDomain}>
            <Label>主域名</Label>
            <Input placeholder="example.com 或 *.example.com" />
          </TextField>
          <div className="grid gap-2">
            <Label htmlFor="certificate-sans">备用域名 SANs</Label>
            <TextArea
              fullWidth
              id="certificate-sans"
              placeholder={'一行一个或用逗号分隔，例如：\nwww.example.com\n*.example.com'}
              rows={4}
              value={sansText}
              onChange={(event) => setSANsText(event.target.value)}
            />
            <Description>后端会统一去重、转小写，并移除与主域名重复的 SAN。</Description>
          </div>
          <div className="grid gap-2">
            <Label>DNS 账号</Label>
            <div className="grid gap-2" role="radiogroup" aria-label="选择 DNS 账号">
              {dnsAccounts.map((account) => (
                <Button
                  key={account.id}
                  className="h-auto justify-start px-4 py-3 text-left"
                  variant={account.id === selectedDNSAccountID ? undefined : "secondary"}
                  onPress={() => setSelectedDNSAccountID(account.id)}
                >
                  <span className="grid gap-1">
                    <span className="font-medium">{account.name}</span>
                    <span className="text-xs text-slate-500">{account.provider} · {account.accessKeyMasked || "未填写 AccessKey"}</span>
                  </span>
                </Button>
              ))}
            </div>
          </div>
          <Button isDisabled={dnsAccounts.length === 0} isPending={pending} type="submit">
            {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}创建证书申请</>}
          </Button>
        </Form>

        <div className="grid gap-3">
          {applications.length === 0 ? (
            <EmptyState title="还没有证书申请" description="创建后会在这里看到申请记录。" />
          ) : (
            applications.map((application) => (
              <div key={application.id} className="rounded-3xl border border-slate-200/80 bg-slate-50/80 p-4">
                <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <div className="font-medium text-slate-950">{application.primaryDomain}</div>
                      <StatusPill>{application.status}</StatusPill>
                    </div>
                    <div className="mt-2 text-sm text-slate-500">
                      DNS：{application.dnsAccountName || application.dnsAccountId} · Challenge：{application.challengeMode}
                    </div>
                    {application.sans.length > 0 ? (
                      <div className="mt-3 flex flex-wrap gap-2">
                        {application.sans.map((domain) => (
                          <span key={domain} className="rounded-full bg-white px-3 py-1 text-xs text-slate-600 shadow-sm">{domain}</span>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  <div className="text-xs text-slate-400">{formatDateTime(application.createdAt)}</div>
                </div>
              </div>
            ))
          )}
        </div>
      </Card.Content>
    </Card>
  );
}

function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-3xl border border-dashed border-slate-300 bg-white/60 p-6 text-center">
      <div className="font-medium text-slate-950">{title}</div>
      <div className="mt-2 text-sm leading-6 text-slate-500">{description}</div>
    </div>
  );
}

function StatusPill({ children }: { children: React.ReactNode }) {
  return <span className="rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700">{children}</span>;
}

function SummaryCard({ title, value, description }: { title: string; value: string; description: string }) {
  return (
    <Card className="p-5">
      <Card.Header>
        <Card.Description>{title}</Card.Description>
        <Card.Title>{value}</Card.Title>
      </Card.Header>
      <Card.Content className="text-sm leading-6 text-slate-500">{description}</Card.Content>
    </Card>
  );
}

function InlineAlert({ status, title, description }: { status: "accent" | "danger"; title: string; description?: string }) {
  return (
    <Alert status={status}>
      <Alert.Indicator />
      <Alert.Content>
        <Alert.Title>{title}</Alert.Title>
        {description ? <Alert.Description>{description}</Alert.Description> : null}
      </Alert.Content>
    </Alert>
  );
}

function parseDomainList(value: string) {
  return value
    .split(/[\s,]+/)
    .map((domain) => domain.trim())
    .filter(Boolean);
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

function normalizeRoute(pathname: string): Route {
  return routeSet.has(pathname as Route) ? (pathname as Route) : "/";
}

function errorMessage(err: unknown, fallback: string) {
  if (err instanceof ApiError || err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}
