import type { ReactNode } from "react";
import { Inbox, AlertTriangle, RefreshCw } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { errorKey } from "@/shared/api/client";

export function EmptyState({
  title,
  body,
  action,
}: {
  title: string;
  body?: string;
  action?: ReactNode;
}) {
  return (
    <div className="state-box">
      <Inbox size={26} aria-hidden />
      <div className="state-title">{title}</div>
      {body && <p className="faint">{body}</p>}
      {action}
    </div>
  );
}

export function ErrorState({
  error,
  onRetry,
}: {
  error: unknown;
  onRetry?: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className="state-box" role="alert">
      <AlertTriangle size={26} aria-hidden />
      <div className="state-title">{t("common.error.title")}</div>
      <p className="faint">{t(errorKey(error))}</p>
      {onRetry && (
        <button className="btn btn-secondary" onClick={onRetry}>
          <RefreshCw size={14} aria-hidden />
          {t("common.retry")}
        </button>
      )}
    </div>
  );
}

export function SkeletonRows({ rows = 4 }: { rows?: number }) {
  return (
    <div className="stack" style={{ padding: 16 }} aria-hidden>
      {Array.from({ length: rows }, (_, i) => (
        <div
          key={i}
          className="skeleton"
          style={{ height: 18, width: `${88 - (i % 3) * 14}%` }}
        />
      ))}
    </div>
  );
}
