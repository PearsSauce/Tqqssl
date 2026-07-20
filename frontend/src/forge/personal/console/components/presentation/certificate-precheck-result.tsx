import type { CertificatePrecheck } from "../../../../../types";

export function CertificatePrecheckResult({ precheck }: { precheck: CertificatePrecheck }) {
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
