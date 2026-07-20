import type { CertificateApplication, CertificateAuthorization } from "../../../../../types";

import { StatusPill } from "./shared";

export function CertificateAuthorizationDetails({ authorizations }: { authorizations: CertificateAuthorization[] }) {
  return (
    <div className="mt-3 grid gap-3">
      {authorizations.map((authorization) => (
        <div key={authorization.url} className="rounded-2xl border border-blue-100 bg-blue-50/60 p-3 text-xs leading-5 text-slate-600">
          <div className="flex flex-wrap items-center gap-2">
            <div className="font-medium text-slate-800">{authorization.domain}</div>
            <StatusPill>{authorization.status}</StatusPill>
            {authorization.wildcard ? <StatusPill>泛域名</StatusPill> : null}
          </div>
          {authorization.dns01 ? (
            <div className="mt-3 grid gap-2">
              <DNSRecordRow label="记录名" value={authorization.dns01.recordName} />
              <DNSRecordRow label="记录类型" value={authorization.dns01.recordType} />
              <DNSRecordRow label="记录值" value={authorization.dns01.recordValue} />
            </div>
          ) : (
            <div className="mt-2 text-amber-700">该 authorization 未返回 dns-01 challenge。</div>
          )}
        </div>
      ))}
    </div>
  );
}

function DNSRecordRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 rounded-xl bg-white/80 px-3 py-2">
      <div className="font-medium text-slate-700">{label}</div>
      <div className="break-all text-slate-600">{value}</div>
    </div>
  );
}

export function CertificateOrderDetails({ application }: { application: CertificateApplication }) {
  const rows = [
    { label: "Order URL", value: application.orderUrl },
    { label: "Finalize URL", value: application.finalizeUrl },
    { label: "Authorization", value: application.authorizationUrls?.join("\n") }
  ].filter((row) => row.value);
  return (
    <div className="mt-3 grid gap-2 rounded-2xl border border-emerald-100 bg-emerald-50/70 p-3 text-xs leading-5 text-slate-600">
      {rows.map((row) => (
        <div key={row.label} className="grid gap-1">
          <div className="font-medium text-slate-700">{row.label}</div>
          <div className="whitespace-pre-wrap break-all">{row.value}</div>
        </div>
      ))}
    </div>
  );
}
