import { Button, Card, Spinner } from "@heroui/react";
import { useState } from "react";

import { apiRequest } from "../../../../../api";
import { errorMessage } from "../../../../../lib/utils";
import type { ACMEAccountRegistration, ACMEDirectoryCheck, ACMEStatus } from "../../../../../types";

import { InlineAlert, StatusPill } from "./shared";

export function ACMEStatusPanel({ status, onUpdated }: {
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
