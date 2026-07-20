import { Button, Card, Description, Form, Input, Label, Spinner, TextArea, TextField } from "@heroui/react";
import { useEffect, useState } from "react";

import { apiRequest } from "../../../../../api";
import { errorMessage, formatDateTime, parseDomainList } from "../../../../../lib/utils";
import type { ACMEStatus, CertificateApplication, CertificateAuthorization, CertificatePrecheck, DNSAccount } from "../../../../../types";

import { CertificateAuthorizationDetails, CertificateOrderDetails } from "./certificate-application-details";
import { CertificatePrecheckResult } from "./certificate-precheck-result";
import { EmptyState, InlineAlert, StatusPill } from "./shared";

export function CertificateApplicationsPanel({ applications, acmeStatus, dnsAccounts, onCreated, onUpdated, onDeleted }: {
  applications: CertificateApplication[];
  acmeStatus: ACMEStatus | null;
  dnsAccounts: DNSAccount[];
  onCreated: (application: CertificateApplication) => void;
  onUpdated: (application: CertificateApplication) => void;
  onDeleted: (applicationID: string) => void;
}) {
  const [primaryDomain, setPrimaryDomain] = useState("");
  const [sansText, setSANsText] = useState("");
  const [selectedDNSAccountID, setSelectedDNSAccountID] = useState("");
  const [pending, setPending] = useState(false);
  const [prechecking, setPrechecking] = useState(false);
  const [orderingID, setOrderingID] = useState("");
  const [loadingAuthorizationsID, setLoadingAuthorizationsID] = useState("");
  const [authorizationsByApplicationID, setAuthorizationsByApplicationID] = useState<Record<string, CertificateAuthorization[]>>({});
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

  async function createACMEOrder(applicationID: string) {
    setOrderingID(applicationID);
    setError("");
    try {
      const application = await apiRequest<CertificateApplication>(`/certificates/applications/${applicationID}/acme/order`, { method: "POST" });
      onUpdated(application);
    } catch (err) {
      setError(errorMessage(err, "创建 ACME order 失败"));
    } finally {
      setOrderingID("");
    }
  }

  async function loadACMEAuthorizations(applicationID: string) {
    setLoadingAuthorizationsID(applicationID);
    setError("");
    try {
      const authorizations = await apiRequest<CertificateAuthorization[]>(`/certificates/applications/${applicationID}/acme/authorizations`);
      setAuthorizationsByApplicationID((current) => ({ ...current, [applicationID]: authorizations }));
    } catch (err) {
      setError(errorMessage(err, "读取 ACME 授权失败"));
    } finally {
      setLoadingAuthorizationsID("");
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
          {!acmeStatus?.accountRegistered ? <InlineAlert status="accent" title="ACME 账号未注册" description="证书申请可以先保存；创建 ACME order 前需要先注册 ACME 账号。" /> : null}
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
                      {application.orderStatus ? <StatusPill>order {application.orderStatus}</StatusPill> : null}
                    </div>
                    <div className="mt-2 text-sm text-slate-500">
                      DNS：{application.dnsAccountName || application.dnsAccountId} · Challenge：{application.challengeMode}
                    </div>
                    {application.orderUrl ? <CertificateOrderDetails application={application} /> : null}
                    {authorizationsByApplicationID[application.id] ? (
                      <CertificateAuthorizationDetails authorizations={authorizationsByApplicationID[application.id]} />
                    ) : null}
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
                      isDisabled={!acmeStatus?.accountRegistered || Boolean(application.orderUrl)}
                      isPending={orderingID === application.id}
                      variant="secondary"
                      onPress={() => void createACMEOrder(application.id)}
                    >
                      {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}{application.orderUrl ? "ACME Order 已创建" : "创建 ACME Order"}</>}
                    </Button>
                    <Button
                      isDisabled={!application.orderUrl}
                      isPending={loadingAuthorizationsID === application.id}
                      variant="secondary"
                      onPress={() => void loadACMEAuthorizations(application.id)}
                    >
                      {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}查看 DNS-01 记录</>}
                    </Button>
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
