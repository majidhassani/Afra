import { useQuery } from "@tanstack/react-query";
import { RefreshCw, CircleCheck, CircleX } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { healthApi } from "@/shared/api/endpoints";
import { env } from "@/shared/config/env";
import { lastApiError } from "@/shared/api/client";
import { useAuthStore } from "@/features/auth/authStore";
import { Button } from "@/shared/ui/Button";

function StatusChip({ ok, label }: { ok: boolean | undefined; label: string }) {
  return (
    <span
      className="chip"
      style={{
        color:
          ok === undefined
            ? "var(--text-faint)"
            : ok
              ? "var(--accent-mission)"
              : "var(--accent-danger)",
      }}
    >
      {ok === undefined ? null : ok ? (
        <CircleCheck size={12} aria-hidden />
      ) : (
        <CircleX size={12} aria-hidden />
      )}
      {label}
    </span>
  );
}

export function DiagnosticsPage() {
  const { t } = useI18n();
  const session = useAuthStore((s) => s.session);

  const health = useQuery({
    queryKey: ["health"],
    queryFn: () => healthApi.check("/health"),
  });
  const ready = useQuery({
    queryKey: ["ready"],
    queryFn: () => healthApi.check("/ready"),
  });

  const rows: Array<[string, React.ReactNode]> = [
    [
      t("diag.health"),
      <span key="h" className="row">
        <StatusChip ok={health.data?.ok} label={health.data?.ok ? "OK" : "DOWN"} />
        <span className="faint mono-num">
          {t("diag.latency")}: {health.data ? `${health.data.latencyMs}ms` : "—"}
        </span>
      </span>,
    ],
    [
      t("diag.ready"),
      <StatusChip
        key="r"
        ok={ready.data?.ok}
        label={ready.data?.ok ? "OK" : "DOWN"}
      />,
    ],
    [t("diag.apiBase"), <code key="a">{env.apiBaseUrl}</code>],
    [
      t("diag.auth"),
      session ? (
        <span key="s" className="chip chip-mission">
          {t("diag.authed")}: {session.user.email}
        </span>
      ) : (
        <span key="s" className="chip">
          {t("diag.anonymous")}
        </span>
      ),
    ],
    [t("diag.version"), <code key="v">{env.appVersion}</code>],
    [
      t("diag.mocks"),
      <span key="m" className="chip">
        {env.enableMocks ? t("diag.on") : t("diag.off")}
      </span>,
    ],
    [
      t("diag.lastError"),
      lastApiError.current ? (
        <code key="e" style={{ color: "var(--accent-danger)" }}>
          {lastApiError.current.code}: {lastApiError.current.message}
        </code>
      ) : (
        <span key="e" className="faint">
          {t("diag.none")}
        </span>
      ),
    ],
  ];

  return (
    <div className="page" style={{ maxWidth: 720 }}>
      <header className="page-header">
        <h1>{t("diag.title")}</h1>
        <Button
          variant="secondary"
          onClick={() => {
            void health.refetch();
            void ready.refetch();
          }}
        >
          <RefreshCw size={14} aria-hidden />
          {t("diag.check")}
        </Button>
      </header>

      <div className="panel item-list">
        {rows.map(([label, value]) => (
          <div key={label} className="item-row">
            <span className="grow sub">{label}</span>
            {value}
          </div>
        ))}
      </div>
    </div>
  );
}
