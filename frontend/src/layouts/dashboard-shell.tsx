import { Button, Spinner } from "@heroui/react";
import type { ReactNode } from "react";

export type DashboardNavigationItem<TSection extends string = string> = {
  id: TSection;
  label: string;
  description: string;
  badge?: string;
};

type DashboardShellProps<TSection extends string> = {
  activeItemID: TSection;
  children: ReactNode;
  createdAt: string;
  description: string;
  eyebrow: string;
  logoutPending: boolean;
  navigationItems: DashboardNavigationItem<TSection>[];
  title: string;
  userEmail: string;
  userName: string;
  onLogout: () => void;
  onNavigate: (itemID: TSection) => void;
};

export function DashboardShell<TSection extends string>({
  activeItemID,
  children,
  createdAt,
  description,
  eyebrow,
  logoutPending,
  navigationItems,
  title,
  userEmail,
  userName,
  onLogout,
  onNavigate
}: DashboardShellProps<TSection>) {
  return (
    <main className="min-h-screen px-4 py-5 sm:px-6 lg:px-8">
      <div className="mx-auto grid max-w-7xl gap-5 lg:grid-cols-[18rem_minmax(0,1fr)]">
        <aside className="hidden lg:block">
          <div className="sticky top-5 flex min-h-[calc(100vh-2.5rem)] flex-col rounded-[2rem] border border-white/70 bg-slate-950 p-4 text-white shadow-2xl shadow-slate-950/10">
            <BrandBlock />
            <nav className="mt-8 grid gap-2" aria-label="控制台侧边栏导航">
              {navigationItems.map((item) => (
                <NavigationButton
                  key={item.id}
                  isActive={item.id === activeItemID}
                  item={item}
                  onPress={() => onNavigate(item.id)}
                />
              ))}
            </nav>
            <div className="mt-auto rounded-3xl border border-white/10 bg-white/10 p-4 text-sm text-white/75">
              <div className="font-medium text-white">个人版边界</div>
              <div className="mt-2 leading-6">单管理员、本地账号、API 优先；不包含 SSO/OIDC、Agent、订阅、支付、公告和兑换。</div>
            </div>
          </div>
        </aside>

        <div className="grid min-w-0 gap-5">
          <section className="rounded-[2rem] border border-white/70 bg-white/85 p-4 shadow-xl shadow-blue-950/5 backdrop-blur sm:p-6">
            <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2 text-sm text-slate-500">
                  <span>{eyebrow}</span>
                  <span className="hidden size-1 rounded-full bg-slate-300 sm:inline-block" aria-hidden="true" />
                  <span className="break-all">{userEmail}</span>
                </div>
                <h1 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950 sm:text-4xl">{title}</h1>
                <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-500 sm:text-base">{description}</p>
              </div>
              <div className="flex flex-col gap-3 rounded-3xl border border-slate-200/80 bg-slate-50/80 p-3 sm:flex-row sm:items-center sm:justify-between xl:min-w-80">
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium text-slate-950">{userName}</div>
                  <div className="mt-1 text-xs leading-5 text-slate-500">账号创建于 {createdAt}</div>
                </div>
                <Button variant="secondary" isPending={logoutPending} onPress={onLogout}>
                  {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}退出登录</>}
                </Button>
              </div>
            </div>
          </section>

          <nav className="scrollbar-none flex gap-2 overflow-x-auto rounded-[1.5rem] border border-white/70 bg-white/75 p-2 shadow-lg shadow-blue-950/5 backdrop-blur lg:hidden" aria-label="控制台移动导航">
            {navigationItems.map((item) => (
              <Button
                key={item.id}
                className="h-auto shrink-0 px-4 py-3 text-left"
                variant={item.id === activeItemID ? undefined : "secondary"}
                onPress={() => onNavigate(item.id)}
              >
                <span className="grid gap-1">
                  <span className="font-medium">{item.label}</span>
                  {item.badge ? <span className="text-xs opacity-70">{item.badge}</span> : null}
                </span>
              </Button>
            ))}
          </nav>

          <section className="grid min-w-0 gap-5" aria-live="polite">
            {children}
          </section>
        </div>
      </div>
    </main>
  );
}

function BrandBlock() {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/10 p-4">
      <div className="flex items-center gap-3">
        <div className="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-white text-lg font-semibold text-slate-950 shadow-xl shadow-black/20">T</div>
        <div className="min-w-0">
          <div className="truncate font-semibold">Tqqssl 个人版</div>
          <div className="mt-1 text-xs text-white/55">Personal SSL Console</div>
        </div>
      </div>
    </div>
  );
}

function NavigationButton<TSection extends string>({ item, isActive, onPress }: {
  item: DashboardNavigationItem<TSection>;
  isActive: boolean;
  onPress: () => void;
}) {
  return (
    <Button
      className={`h-auto justify-start px-4 py-3 text-left ${isActive ? "bg-white text-slate-950 shadow-lg shadow-black/20" : "text-white/82 hover:bg-white/10"}`}
      variant="ghost"
      aria-current={isActive ? "page" : undefined}
      onPress={onPress}
    >
      <span className="grid min-w-0 flex-1 gap-1">
        <span className="flex items-center justify-between gap-3">
          <span className="font-medium">{item.label}</span>
          {item.badge ? (
            <span className={`rounded-full px-2 py-0.5 text-[0.68rem] font-medium ${isActive ? "bg-slate-100 text-slate-600" : "bg-white/10 text-white/70"}`}>
              {item.badge}
            </span>
          ) : null}
        </span>
        <span className={`text-xs leading-5 ${isActive ? "text-slate-500" : "text-white/55"}`}>{item.description}</span>
      </span>
    </Button>
  );
}
