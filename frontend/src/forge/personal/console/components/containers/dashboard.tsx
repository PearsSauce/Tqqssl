import { Card, Spinner } from "@heroui/react";

import { DashboardShell } from "../../../../../layouts/dashboard-shell";
import type { User } from "../../../../../types";
import { useDashboardPage } from "../../hooks/use-dashboard-page";
import { DashboardSectionContent } from "../presentation/dashboard-section-content";
import { InlineAlert } from "../presentation/shared";

export function Dashboard({ user, onLogout }: { user: User; onLogout: () => Promise<void> }) {
  const page = useDashboardPage({ user, onLogout });

  return (
    <DashboardShell
      activeItemID={page.activeSection}
      createdAt={page.createdAt}
      description={page.activeSectionMeta.description}
      eyebrow="个人版控制台"
      logoutPending={page.logoutPending}
      navigationItems={page.navigationItems}
      title={page.activeSectionMeta.title}
      userEmail={user.email}
      userName={user.username}
      onLogout={page.logout}
      onNavigate={page.navigateDashboardSection}
    >
      {page.resourcesError ? <InlineAlert status="danger" title={page.resourcesError} /> : null}
      {page.resourcesLoading ? (
        <Card className="items-center gap-3 p-6 text-center">
          <Spinner />
          <Card.Description>正在加载 DNS 账号与证书申请记录。</Card.Description>
        </Card>
      ) : (
        <DashboardSectionContent
          activeSection={page.activeSection}
          acmeStatus={page.acmeStatus}
          applications={page.applications}
          dnsAccounts={page.dnsAccounts}
          onACMEStatusUpdated={page.setACMEStatus}
          onCertificateApplicationCreated={(application) => page.setApplications((current) => [application, ...current])}
          onCertificateApplicationDeleted={(applicationID) => page.setApplications((current) => current.filter((application) => application.id !== applicationID))}
          onCertificateApplicationUpdated={(application) => page.setApplications((current) => current.map((item) => item.id === application.id ? application : item))}
          onDNSAccountCreated={(account) => page.setDNSAccounts((current) => [account, ...current])}
          onDNSAccountDeleted={(accountID) => page.setDNSAccounts((current) => current.filter((account) => account.id !== accountID))}
          onDNSAccountUpdated={(account) => page.setDNSAccounts((current) => current.map((item) => item.id === account.id ? account : item))}
        />
      )}
    </DashboardShell>
  );
}
