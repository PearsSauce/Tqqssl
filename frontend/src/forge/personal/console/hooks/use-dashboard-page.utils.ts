import type { DashboardNavigationItem } from "../../../../layouts/dashboard-shell";
import type { ACMEStatus, CertificateApplication, DNSAccount } from "../../../../types";

export type DashboardSection = "overview" | "acme" | "dns" | "certificates";

const dashboardSectionSet = new Set<DashboardSection>(["overview", "acme", "dns", "certificates"]);

const baseDashboardNavigationItems: Array<Omit<DashboardNavigationItem<DashboardSection>, "badge">> = [
  { id: "overview", label: "总览", description: "资源概览与实现边界" },
  { id: "acme", label: "ACME", description: "账号就绪、目录检查和注册" },
  { id: "dns", label: "DNS 账号", description: "本地加密保存 DNS 凭据" },
  { id: "certificates", label: "证书申请", description: "DNS-01 申请与记录预览" }
];

export const dashboardSectionMeta: Record<DashboardSection, { title: string; description: string }> = {
  overview: {
    title: "个人版控制台",
    description: "查看个人版 DNS、证书申请和 ACME 配置概览，并确认当前实现边界。"
  },
  acme: {
    title: "ACME 配置",
    description: "检查签发前置条件、验证 ACME directory，并注册本地 ACME 账号。"
  },
  dns: {
    title: "DNS 账号管理",
    description: "维护 DNS API 凭据。SecretKey 只加密写入本地数据文件，不会从 API 返回。"
  },
  certificates: {
    title: "证书申请",
    description: "创建 DNS-01 证书申请、发起 ACME order，并查看需要写入的 TXT 记录。"
  }
};

export function getDashboardSectionFromLocation(): DashboardSection {
  const section = window.location.hash.replace(/^#/, "");
  return dashboardSectionSet.has(section as DashboardSection) ? (section as DashboardSection) : "overview";
}

export function buildDashboardSectionURL(nextSection: DashboardSection) {
  return nextSection === "overview" ? "/" : `/#${nextSection}`;
}

export function buildDashboardNavigationItems({ acmeStatus, applications, dnsAccounts }: {
  acmeStatus: ACMEStatus | null;
  applications: CertificateApplication[];
  dnsAccounts: DNSAccount[];
}): DashboardNavigationItem<DashboardSection>[] {
  return baseDashboardNavigationItems.map((item) => {
    if (item.id === "acme") {
      return { ...item, badge: acmeStatus?.ready ? "就绪" : "待配置" };
    }
    if (item.id === "dns") {
      return { ...item, badge: `${dnsAccounts.length} 个` };
    }
    if (item.id === "certificates") {
      return { ...item, badge: `${applications.length} 条` };
    }
    return item;
  });
}
