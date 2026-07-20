import type { ACMEStatus, CertificateApplication, DNSAccount } from "../../../../../types";
import type { DashboardSection } from "../../hooks/use-dashboard-page.utils";

import { ACMEStatusPanel } from "./acme-status-panel";
import { CertificateApplicationsPanel } from "./certificate-applications-panel";
import { DashboardOverview } from "./dashboard-overview";
import { DNSAccountsPanel } from "./dns-accounts-panel";

export type DashboardSectionContentProps = {
  activeSection: DashboardSection;
  acmeStatus: ACMEStatus | null;
  applications: CertificateApplication[];
  dnsAccounts: DNSAccount[];
  onACMEStatusUpdated: (status: ACMEStatus) => void;
  onCertificateApplicationCreated: (application: CertificateApplication) => void;
  onCertificateApplicationDeleted: (applicationID: string) => void;
  onCertificateApplicationUpdated: (application: CertificateApplication) => void;
  onDNSAccountCreated: (account: DNSAccount) => void;
  onDNSAccountDeleted: (accountID: string) => void;
  onDNSAccountUpdated: (account: DNSAccount) => void;
};

export function DashboardSectionContent({
  activeSection,
  acmeStatus,
  applications,
  dnsAccounts,
  onACMEStatusUpdated,
  onCertificateApplicationCreated,
  onCertificateApplicationDeleted,
  onCertificateApplicationUpdated,
  onDNSAccountCreated,
  onDNSAccountDeleted,
  onDNSAccountUpdated
}: DashboardSectionContentProps) {
  if (activeSection === "acme") {
    return <ACMEStatusPanel status={acmeStatus} onUpdated={onACMEStatusUpdated} />;
  }

  if (activeSection === "dns") {
    return (
      <DNSAccountsPanel
        accounts={dnsAccounts}
        onCreated={onDNSAccountCreated}
        onDeleted={onDNSAccountDeleted}
        onUpdated={onDNSAccountUpdated}
      />
    );
  }

  if (activeSection === "certificates") {
    return (
      <CertificateApplicationsPanel
        acmeStatus={acmeStatus}
        applications={applications}
        dnsAccounts={dnsAccounts}
        onCreated={onCertificateApplicationCreated}
        onDeleted={onCertificateApplicationDeleted}
        onUpdated={onCertificateApplicationUpdated}
      />
    );
  }

  return <DashboardOverview acmeStatus={acmeStatus} applications={applications} dnsAccounts={dnsAccounts} />;
}
