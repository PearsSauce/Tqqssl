import { useEffect, useMemo, useState } from "react";

import { apiRequest } from "../../../../api";
import { errorMessage, formatDateTime } from "../../../../lib/utils";
import type { ACMEStatus, CertificateApplication, DNSAccount, User } from "../../../../types";

import {
  buildDashboardNavigationItems,
  buildDashboardSectionURL,
  dashboardSectionMeta,
  getDashboardSectionFromLocation,
  type DashboardSection
} from "./use-dashboard-page.utils";

type UseDashboardPageOptions = {
  user: User;
  onLogout: () => Promise<void>;
};

export function useDashboardPage({ user, onLogout }: UseDashboardPageOptions) {
  const createdAt = useMemo(() => formatDateTime(user.createdAt), [user.createdAt]);
  const [activeSection, setActiveSection] = useState<DashboardSection>(() => getDashboardSectionFromLocation());
  const [logoutPending, setLogoutPending] = useState(false);
  const [dnsAccounts, setDNSAccounts] = useState<DNSAccount[]>([]);
  const [applications, setApplications] = useState<CertificateApplication[]>([]);
  const [acmeStatus, setACMEStatus] = useState<ACMEStatus | null>(null);
  const [resourcesLoading, setResourcesLoading] = useState(true);
  const [resourcesError, setResourcesError] = useState("");

  useEffect(() => {
    function syncSectionFromLocation() {
      setActiveSection(getDashboardSectionFromLocation());
    }
    window.addEventListener("hashchange", syncSectionFromLocation);
    window.addEventListener("popstate", syncSectionFromLocation);
    return () => {
      window.removeEventListener("hashchange", syncSectionFromLocation);
      window.removeEventListener("popstate", syncSectionFromLocation);
    };
  }, []);

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

  const navigationItems = useMemo(
    () => buildDashboardNavigationItems({ acmeStatus, applications, dnsAccounts }),
    [acmeStatus, applications, dnsAccounts]
  );
  const activeSectionMeta = dashboardSectionMeta[activeSection];

  function navigateDashboardSection(nextSection: DashboardSection) {
    const nextURL = buildDashboardSectionURL(nextSection);
    if (`${window.location.pathname}${window.location.hash}` !== nextURL) {
      window.history.pushState({}, "", nextURL);
    }
    setActiveSection(nextSection);
  }

  function logout() {
    setLogoutPending(true);
    void onLogout().finally(() => setLogoutPending(false));
  }

  return {
    acmeStatus,
    activeSection,
    activeSectionMeta,
    applications,
    createdAt,
    dnsAccounts,
    logout,
    logoutPending,
    navigationItems,
    navigateDashboardSection,
    resourcesError,
    resourcesLoading,
    setACMEStatus,
    setApplications,
    setDNSAccounts
  };
}
