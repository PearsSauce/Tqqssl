import { Alert, Button, Card, Description, FieldError, Form, Input, Label, Spinner, TextField } from "@heroui/react";
import { useEffect, useMemo, useState } from "react";

import { ApiError, apiRequest } from "./api";
import type { RegisterOptions, User } from "./types";

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
  const createdAt = useMemo(() => new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(user.createdAt)), [user.createdAt]);
  const [pending, setPending] = useState(false);
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
          <SummaryCard title="账号模式" value="单管理员" description="没有多用户、租户和 SSO 绑定。" />
          <SummaryCard title="部署模式" value="API Only" description="不包含 Agent，本地证书能力后续按 API 实现。" />
          <SummaryCard title="商业模块" value="未启用" description="没有订阅、支付、公告和兑换。" />
        </div>

        <Card className="p-6">
          <Card.Header>
            <Card.Title>下一步</Card.Title>
            <Card.Description>基础认证已经就绪，后续可以继续添加 DNS 账号、证书申请和部署 API。</Card.Description>
          </Card.Header>
          <Card.Content>
            <InlineAlert status="accent" title="当前是干净个人版起点" description="本仓库没有从商业版复制业务代码。后续每个模块都按个人版需求重新实现。" />
          </Card.Content>
        </Card>
      </section>
    </main>
  );
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

function normalizeRoute(pathname: string): Route {
  return routeSet.has(pathname as Route) ? (pathname as Route) : "/";
}

function errorMessage(err: unknown, fallback: string) {
  if (err instanceof ApiError || err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}
