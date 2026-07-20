import { ApiError } from "../api";

export function errorMessage(err: unknown, fallback: string) {
  if (err instanceof ApiError || err instanceof Error) {
    return err.message || fallback;
  }
  return fallback;
}

export function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

export function parseDomainList(value: string) {
  return value
    .split(/[\s,]+/)
    .map((domain) => domain.trim())
    .filter(Boolean);
}
