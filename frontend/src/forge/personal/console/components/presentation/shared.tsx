import { Alert, Card } from "@heroui/react";
import type { ReactNode } from "react";

export function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-3xl border border-dashed border-slate-300 bg-white/60 p-6 text-center">
      <div className="font-medium text-slate-950">{title}</div>
      <div className="mt-2 text-sm leading-6 text-slate-500">{description}</div>
    </div>
  );
}

export function StatusPill({ children }: { children: ReactNode }) {
  return <span className="rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700">{children}</span>;
}

export function SummaryCard({ title, value, description }: { title: string; value: string; description: string }) {
  return (
    <Card className="p-5">
      <Card.Header>
        <Card.Description>{title}</Card.Description>
        <Card.Title>{value}</Card.Title>
      </Card.Header>
      <Card.Content className="text-sm leading-6 text-slate-500">{description}</Card.Content>
    </Card>
  );
}

export function InlineAlert({ status, title, description }: { status: "accent" | "danger"; title: string; description?: string }) {
  return (
    <Alert status={status}>
      <Alert.Indicator />
      <Alert.Content>
        <Alert.Title>{title}</Alert.Title>
        {description ? <Alert.Description>{description}</Alert.Description> : null}
      </Alert.Content>
    </Alert>
  );
}
