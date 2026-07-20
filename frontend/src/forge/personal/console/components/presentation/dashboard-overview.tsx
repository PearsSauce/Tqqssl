import { Card } from "@heroui/react";

import type { ACMEStatus, CertificateApplication, DNSAccount } from "../../../../../types";

import { InlineAlert, SummaryCard } from "./shared";

export function DashboardOverview({ acmeStatus, applications, dnsAccounts }: {
  acmeStatus: ACMEStatus | null;
  applications: CertificateApplication[];
  dnsAccounts: DNSAccount[];
}) {
  return (
    <>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard title="DNS 账号" value={`${dnsAccounts.length} 个`} description="本地加密保存 DNS API 凭据，接口响应不返回 SecretKey。" />
        <SummaryCard title="证书申请" value={`${applications.length} 条`} description="建立申请记录后可创建 ACME order，challenge mode 固定为 dns-01。" />
        <SummaryCard title="ACME 就绪" value={acmeStatus?.ready ? "已就绪" : "未就绪"} description="需要账号私钥、目录 URL 和条款确认。" />
        <SummaryCard title="商业模块" value="未启用" description="没有多用户、SSO、Agent、订阅、支付、公告和兑换。" />
      </div>

      <Card className="p-6">
        <Card.Header>
          <Card.Title>实现边界</Card.Title>
          <Card.Description>个人版当前完成 DNS 账号和证书申请基础闭环，后续再接入真实 ACME 签发与 DNS 提供商适配。</Card.Description>
        </Card.Header>
        <Card.Content>
          <InlineAlert status="accent" title="当前是干净个人版实现" description="本仓库没有引入商业化后端、SSO/OIDC、Agent、订阅、支付、公告或兑换模块。" />
        </Card.Content>
      </Card>
    </>
  );
}
