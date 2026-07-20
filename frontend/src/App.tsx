import { Alert, Button, Card, Description, FieldError, Form, Input, Label, Spinner, TextArea, TextField } from "@heroui/react";
import { useEffect, useMemo, useState } from "react";

import { ApiError, apiRequest } from "./api";
import type { ACMEAccountRegistration, ACMEDirectoryCheck, ACMEStatus, CertificateApplication, CertificatePrecheck, DNSAccount, RegisterOptions, User } from "./types";

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
  const [acmeStatus, setACMEStatus] = useState<ACMEStatus | null>(null);
  const [resourcesLoading, setResourcesLoading] = useState(true);
  const [resourcesError, setResourcesError] = useState("");

  useEffect(() => {
    let cancelled = false;
    async function loadResources() {
      setResourcesLoading(true);
      setResourcesError("");
      try {
        const [nextDNSAccounts, nextApplications, nextACMEStatus] = await Promise.all([
          apiRequest<DNSAccount[]>("/dns-accounts"),
          apiRequest<CertificateApplication[]>("/certificates/applications"),
          apiRequest<ACMEStatus>("/acme/status")
        ]);
        if (!cancelled) {
          setDNSAccounts(nextDNSAccounts);
          setApplications(nextApplications);
          setACMEStatus(nextACMEStatus);
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

        <div className="grid gap-4 md:grid-cols-4">
          <SummaryCard title="DNS 账号" value={`${dnsAccounts.length} 个`} description="本地加密保存 DNS API 凭据，接口响应不返回 SecretKey。" />
          <SummaryCard title="证书申请" value={`${applications.length} 条`} description="当前建立申请记录，challenge mode 固定为 dns-01。" />
          <SummaryCard title="ACME 就绪" value={acmeStatus?.ready ? "已就绪" : "未就绪"} description="需要账号私钥、目录 URL 和条款确认。" />
          <SummaryCard title="商业模块" value="未启用" description="没有多用户、SSO、Agent、订阅、支付、公告和兑换。" />
        </div>

        {resourcesError ? <InlineAlert status="danger" title={resourcesError} /> : null}
        {resourcesLoading ? (
          <Card className="items-center gap-3 p-6 text-center">
            <Spinner />
            <Card.Description>正在加载 DNS 账号与证书申请记录。</Card.Description>
          </Card>
        ) : (
          <div className="grid gap-6">
            <ACMEStatusPanel status={acmeStatus} onUpdated={setACMEStatus} />
            <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
              <DNSAccountsPanel
                accounts={dnsAccounts}
                onCreated={(account) => setDNSAccounts((current) => [account, ...current])}
                onUpdated={(account) => setDNSAccounts((current) => current.map((item) => item.id === account.id ? account : item))}
                onDeleted={(accountID) => setDNSAccounts((current) => current.filter((account) => account.id !== accountID))}
              />
              <CertificateApplicationsPanel
                applications={applications}
                dnsAccounts={dnsAccounts}
                onCreated={(application) => setApplications((current) => [application, ...current])}
                onDeleted={(applicationID) => setApplications((current) => current.filter((application) => application.id !== applicationID))}
              />
            </div>
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

function ACMEStatusPanel({ status, onUpdated }: {
  status: ACMEStatus | null;
  onUpdated: (status: ACMEStatus) => void;
}) {
  const [checking, setChecking] = useState(false);
  const [registering, setRegistering] = useState(false);
  const [checkResult, setCheckResult] = useState<ACMEDirectoryCheck | null>(null);
  const [checkError, setCheckError] = useState("");
  const [registerError, setRegisterError] = useState("");
  const [registerSuccess, setRegisterSuccess] = useState("");

  if (!status) {
    return null;
  }
  const acmeStatus = status;
  const checks = [
    { label: "ACME 账号私钥", passed: acmeStatus.accountKeyReady, detail: acmeStatus.accountKeyType || "未加载" },
    { label: "目录 URL", passed: Boolean(acmeStatus.directoryUrl), detail: acmeStatus.directoryUrl || "未配置 TQQSSL_ACME_DIRECTORY_URL" },
    { label: "条款确认", passed: acmeStatus.termsAgreed, detail: acmeStatus.termsAgreed ? "已确认" : "未配置 TQQSSL_ACME_TERMS_AGREED=true" }
  ];
  const accountRows = [
    { label: "注册状态", value: acmeStatus.accountRegistered ? "已注册" : "未注册" },
    { label: "账号状态", value: acmeStatus.accountStatus || "-" },
    { label: "联系邮箱", value: acmeStatus.contactEmail || "-" },
    { label: "账号 URL", value: acmeStatus.accountUrl || "-" }
  ];

  async function checkDirectory() {
    setChecking(true);
    setCheckError("");
    setCheckResult(null);
    try {
      const result = await apiRequest<ACMEDirectoryCheck>("/acme/directory/check", { method: "POST" });
      setCheckResult(result);
    } catch (err) {
      setCheckError(errorMessage(err, "ACME directory 检查失败"));
    } finally {
      setChecking(false);
    }
  }

  async function registerAccount() {
    setRegistering(true);
    setRegisterError("");
    setRegisterSuccess("");
    try {
      const result = await apiRequest<ACMEAccountRegistration>("/acme/account/register", { method: "POST" });
      onUpdated({
        ...acmeStatus,
        accountRegistered: result.accountRegistered,
        accountUrl: result.accountUrl,
        accountStatus: result.accountStatus,
        contactEmail: result.contactEmail
      });
      setRegisterSuccess("ACME 账号注册成功，账号状态已写入本地数据文件。");
    } catch (err) {
      setRegisterError(errorMessage(err, "ACME 账号注册失败"));
    } finally {
      setRegistering(false);
    }
  }

  return (
    <Card className="p-6">
      <Card.Header>
        <Card.Title>ACME 就绪状态</Card.Title>
        <Card.Description>检查签发前置条件并注册 ACME 账号，不会输出 ACME 私钥路径或私钥内容。</Card.Description>
      </Card.Header>
      <Card.Content className="grid gap-4">
        <InlineAlert
          status={acmeStatus.ready ? "accent" : "danger"}
          title={acmeStatus.ready ? "ACME 基础配置已就绪" : "ACME 基础配置未完成"}
          description={acmeStatus.ready ? "可以注册 ACME 账号；证书订单流程后续再接入。" : "需要补齐目录 URL、账号私钥和服务条款确认。"}
        />
        <div className="grid gap-3 md:grid-cols-3">
          {checks.map((check) => (
            <div key={check.label} className="rounded-3xl border border-slate-200/80 bg-slate-50/80 p-4">
              <div className="flex items-center justify-between gap-3">
                <div className="font-medium text-slate-950">{check.label}</div>
                <StatusPill>{check.passed ? "通过" : "待配置"}</StatusPill>
              </div>
              <div className="mt-2 break-all text-sm leading-6 text-slate-500">{check.detail}</div>
            </div>
          ))}
        </div>
        <div className="grid gap-3 md:grid-cols-4">
          {accountRows.map((row) => (
            <div key={row.label} className="rounded-3xl border border-slate-200/80 bg-white/80 p-4">
              <div className="text-xs font-medium text-slate-500">{row.label}</div>
              <div className="mt-2 break-all text-sm leading-6 text-slate-700">{row.value}</div>
            </div>
          ))}
        </div>
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          <Button isDisabled={!acmeStatus.directoryUrl} isPending={checking} variant="secondary" onPress={() => void checkDirectory()}>
            {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}检查 ACME Directory</>}
          </Button>
          <Button isDisabled={!acmeStatus.ready || acmeStatus.accountRegistered} isPending={registering} onPress={() => void registerAccount()}>
            {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}{acmeStatus.accountRegistered ? "ACME 账号已注册" : "注册 ACME 账号"}</>}
          </Button>
          <div className="text-xs leading-5 text-slate-500">注册默认使用当前管理员邮箱作为 ACME contact，不发起证书订单。</div>
        </div>
        {checkError ? <InlineAlert status="danger" title={checkError} /> : null}
        {registerError ? <InlineAlert status="danger" title={registerError} /> : null}
        {registerSuccess ? <InlineAlert status="accent" title={registerSuccess} /> : null}
        {checkResult ? <ACMEDirectoryCheckResult result={checkResult} /> : null}
      </Card.Content>
    </Card>
  );
}

function ACMEDirectoryCheckResult({ result }: { result: ACMEDirectoryCheck }) {
  const endpoints = [
    { label: "newNonce", value: result.newNonce },
    { label: "newAccount", value: result.newAccount },
    { label: "newOrder", value: result.newOrder }
  ];
  return (
    <div className="rounded-3xl border border-emerald-100 bg-emerald-50/70 p-4">
      <div className="font-medium text-slate-950">Directory 检查通过</div>
      <div className="mt-1 break-all text-sm text-slate-600">{result.directoryUrl}</div>
      <div className="mt-3 grid gap-2">
        {endpoints.map((endpoint) => (
          <div key={endpoint.label} className="rounded-2xl bg-white/80 px-3 py-2 text-sm shadow-sm">
            <span className="font-medium text-slate-700">{endpoint.label}</span>
            <span className="ml-2 break-all text-slate-500">{endpoint.value}</span>
          </div>
        ))}
      </div>
      {result.termsOfService || result.website ? (
        <div className="mt-3 grid gap-1 text-xs leading-5 text-slate-600">
          {result.termsOfService ? <div className="break-all">服务条款：{result.termsOfService}</div> : null}
          {result.website ? <div className="break-all">CA 网站：{result.website}</div> : null}
        </div>
      ) : null}
      {result.externalAccountRequired ? (
        <div className="mt-3 rounded-2xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs leading-5 text-amber-800">
          该 ACME CA 要求 External Account Binding，当前个人版不会继续注册账号。
        </div>
      ) : null}
      {result.warnings.length > 0 ? (
        <div className="mt-3 grid gap-1 text-xs leading-5 text-amber-700">
          {result.warnings.map((warning) => <div key={warning}>• {warning}</div>)}
        </div>
      ) : null}
    </div>
  );
}

function DNSAccountsPanel({ accounts, onCreated, onUpdated, onDeleted }: {
  accounts: DNSAccount[];
  onCreated: (account: DNSAccount) => void;
  onUpdated: (account: DNSAccount) => void;
  onDeleted: (accountID: string) => void;
}) {
  const [name, setName] = useState("");
  const [provider, setProvider] = useState("alidns");
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [remark, setRemark] = useState("");
  const [editingID, setEditingID] = useState("");
  const [editName, setEditName] = useState("");
  const [editProvider, setEditProvider] = useState("");
  const [editAccessKey, setEditAccessKey] = useState("");
  const [editSecretKey, setEditSecretKey] = useState("");
  const [editRemark, setEditRemark] = useState("");
  const [pending, setPending] = useState(false);
  const [updatingID, setUpdatingID] = useState("");
  const [deletingID, setDeletingID] = useState("");
  const [error, setError] = useState("");
  const editingAccount = useMemo(() => accounts.find((account) => account.id === editingID) || null, [accounts, editingID]);

  useEffect(() => {
    if (editingID && !editingAccount) {
      clearEditForm();
    }
  }, [editingAccount, editingID]);

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

  function startEdit(account: DNSAccount) {
    setEditingID(account.id);
    setEditName(account.name);
    setEditProvider(account.provider);
    setEditAccessKey("");
    setEditSecretKey("");
    setEditRemark(account.remark || "");
    setError("");
  }

  function clearEditForm() {
    setEditingID("");
    setEditName("");
    setEditProvider("");
    setEditAccessKey("");
    setEditSecretKey("");
    setEditRemark("");
  }

  async function submitUpdate() {
    if (!editingID) {
      return;
    }
    setUpdatingID(editingID);
    setError("");
    const body: Record<string, string> = {
      name: editName,
      provider: editProvider,
      remark: editRemark
    };
    if (editAccessKey.trim()) {
      body.accessKey = editAccessKey;
    }
    if (editSecretKey.trim()) {
      body.secretKey = editSecretKey;
    }
    try {
      const account = await apiRequest<DNSAccount>(`/dns-accounts/${editingID}`, {
        method: "PATCH",
        body
      });
      onUpdated(account);
      clearEditForm();
    } catch (err) {
      setError(errorMessage(err, "更新 DNS 账号失败"));
    } finally {
      setUpdatingID("");
    }
  }

  async function deleteAccount(accountID: string) {
    setDeletingID(accountID);
    setError("");
    try {
      await apiRequest<void>(`/dns-accounts/${accountID}`, { method: "DELETE" });
      if (editingID === accountID) {
        clearEditForm();
      }
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
        <Card.Description>保存个人签发证书所需的 DNS API 凭据。SecretKey 会加密后写入本地数据文件，不会从 API 返回。</Card.Description>
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

        {editingAccount ? (
          <Form
            className="grid gap-4 rounded-3xl border border-blue-100 bg-blue-50/60 p-4"
            onSubmit={(event) => {
              event.preventDefault();
              void submitUpdate();
            }}
          >
            <div>
              <div className="font-medium text-slate-950">编辑 DNS 账号</div>
              <div className="mt-1 text-sm text-slate-500">AccessKey 和 SecretKey 留空表示保持原值；填写后会覆盖并加密保存。</div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <TextField isRequired fullWidth name="editDnsName" value={editName} onChange={setEditName}>
                <Label>账号名称</Label>
                <Input placeholder="阿里云主账号" />
              </TextField>
              <TextField isRequired fullWidth name="editProvider" value={editProvider} onChange={setEditProvider}>
                <Label>服务商标识</Label>
                <Input placeholder="alidns / dnspod / cloudflare" />
              </TextField>
            </div>
            <TextField fullWidth name="editAccessKey" value={editAccessKey} onChange={setEditAccessKey}>
              <Label>新 AccessKey</Label>
              <Input autoComplete="off" placeholder={`留空保持当前值：${editingAccount.accessKeyMasked || "未填写"}`} />
            </TextField>
            <TextField fullWidth name="editSecretKey" type="password" value={editSecretKey} onChange={setEditSecretKey}>
              <Label>新 SecretKey / API Token</Label>
              <Input autoComplete="new-password" placeholder="留空保持当前密文，填写则轮换" />
            </TextField>
            <div className="grid gap-2">
              <Label htmlFor="edit-dns-remark">备注</Label>
              <TextArea
                fullWidth
                id="edit-dns-remark"
                maxLength={300}
                rows={3}
                value={editRemark}
                onChange={(event) => setEditRemark(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button isPending={updatingID === editingID} type="submit">
                {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}保存修改</>}
              </Button>
              <Button type="button" variant="secondary" onPress={clearEditForm}>取消</Button>
            </div>
          </Form>
        ) : null}

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
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button variant="secondary" onPress={() => startEdit(account)}>编辑</Button>
                    <Button
                      variant="secondary"
                      isPending={deletingID === account.id}
                      onPress={() => void deleteAccount(account.id)}
                    >
                      删除
                    </Button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </Card.Content>
    </Card>
  );
}

function CertificateApplicationsPanel({ applications, dnsAccounts, onCreated, onDeleted }: {
  applications: CertificateApplication[];
  dnsAccounts: DNSAccount[];
  onCreated: (application: CertificateApplication) => void;
  onDeleted: (applicationID: string) => void;
}) {
  const [primaryDomain, setPrimaryDomain] = useState("");
  const [sansText, setSANsText] = useState("");
  const [selectedDNSAccountID, setSelectedDNSAccountID] = useState("");
  const [pending, setPending] = useState(false);
  const [prechecking, setPrechecking] = useState(false);
  const [precheck, setPrecheck] = useState<CertificatePrecheck | null>(null);
  const [deletingID, setDeletingID] = useState("");
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

  useEffect(() => {
    setPrecheck(null);
  }, [primaryDomain, sansText, selectedDNSAccountID]);

  function certificateRequestBody() {
    return {
      primaryDomain,
      sans: parseDomainList(sansText),
      dnsAccountId: selectedDNSAccountID,
      challengeMode: "dns-01"
    };
  }

  async function precheckApplication() {
    setPrechecking(true);
    setError("");
    try {
      const result = await apiRequest<CertificatePrecheck>("/certificates/applications/precheck", {
        method: "POST",
        body: certificateRequestBody()
      });
      setPrecheck(result);
    } catch (err) {
      setError(errorMessage(err, "证书申请预检查失败"));
    } finally {
      setPrechecking(false);
    }
  }

  async function submit() {
    setPending(true);
    setError("");
    try {
      const application = await apiRequest<CertificateApplication>("/certificates/applications", {
        method: "POST",
        body: certificateRequestBody()
      });
      onCreated(application);
      setPrimaryDomain("");
      setSANsText("");
      setPrecheck(null);
    } catch (err) {
      setError(errorMessage(err, "创建证书申请失败"));
    } finally {
      setPending(false);
    }
  }

  async function deleteApplication(applicationID: string) {
    setDeletingID(applicationID);
    setError("");
    try {
      await apiRequest<void>(`/certificates/applications/${applicationID}`, { method: "DELETE" });
      onDeleted(applicationID);
    } catch (err) {
      setError(errorMessage(err, "删除证书申请失败"));
    } finally {
      setDeletingID("");
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
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button
              isDisabled={dnsAccounts.length === 0}
              isPending={prechecking}
              type="button"
              variant="secondary"
              onPress={() => void precheckApplication()}
            >
              {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}预检查</>}
            </Button>
            <Button isDisabled={dnsAccounts.length === 0} isPending={pending} type="submit">
              {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}创建证书申请</>}
            </Button>
          </div>
        </Form>

        {precheck ? <CertificatePrecheckResult precheck={precheck} /> : null}

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
                  <div className="flex flex-col items-start gap-2 md:items-end">
                    <div className="text-xs text-slate-400">{formatDateTime(application.createdAt)}</div>
                    <Button
                      variant="secondary"
                      isPending={deletingID === application.id}
                      onPress={() => void deleteApplication(application.id)}
                    >
                      删除申请
                    </Button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </Card.Content>
    </Card>
  );
}

function CertificatePrecheckResult({ precheck }: { precheck: CertificatePrecheck }) {
  const domains = [precheck.primaryDomain, ...precheck.sans];
  return (
    <div className="rounded-3xl border border-emerald-100 bg-emerald-50/70 p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="font-medium text-slate-950">预检查通过</div>
          <div className="mt-1 text-sm text-slate-600">
            DNS：{precheck.dnsAccountName}（{precheck.dnsProvider}） · Challenge：{precheck.challengeMode} · 域名数：{precheck.domainCount}
          </div>
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-2">
        {domains.map((domain) => (
          <span key={domain} className="rounded-full bg-white px-3 py-1 text-xs text-slate-600 shadow-sm">{domain}</span>
        ))}
      </div>
      {precheck.warnings.length > 0 ? (
        <div className="mt-3 grid gap-1 text-xs leading-5 text-amber-700">
          {precheck.warnings.map((warning) => <div key={warning}>• {warning}</div>)}
        </div>
      ) : null}
    </div>
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
