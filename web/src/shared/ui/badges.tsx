import { Coins, Signal, SignalZero } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useI18n } from "@/shared/i18n";
import type { Difficulty, MissionStatus } from "@/shared/types/api";
import { walletApi, healthApi } from "@/shared/api/endpoints";
import type { TranslationKey } from "@/shared/i18n/en";

const statusChip: Record<MissionStatus, string> = {
  generating: "chip-rare",
  ready: "chip-ai",
  active: "chip-mission",
  completed: "chip-mission",
  failed: "chip-danger",
  archived: "",
};

export function MissionStatusBadge({ status }: { status: MissionStatus }) {
  const { t } = useI18n();
  const key = `missions.status.${status}` as TranslationKey;
  return <span className={`chip ${statusChip[status] ?? ""}`}>{t(key)}</span>;
}

export function DifficultyBadge({ difficulty }: { difficulty: Difficulty }) {
  const { t } = useI18n();
  const cls = difficulty === "expert" || difficulty === "hard" ? "chip-danger" : "";
  return (
    <span className={`chip ${cls}`}>
      {t(`difficulty.${difficulty}` as TranslationKey)}
    </span>
  );
}

export function CostBadge({ coins }: { coins: number }) {
  const { t } = useI18n();
  if (coins <= 0) return <span className="chip">{t("common.free")}</span>;
  return (
    <span className="chip chip-wallet mono-num">
      <Coins size={12} aria-hidden />
      {coins}
    </span>
  );
}

export function WalletChip() {
  const { t } = useI18n();
  const { data: wallet } = useQuery({
    queryKey: ["wallet"],
    queryFn: walletApi.get,
    staleTime: 15_000,
  });
  return (
    <span
      className="chip chip-wallet mono-num"
      title={t("wallet.balance")}
      aria-label={t("wallet.balance")}
    >
      <Coins size={13} aria-hidden />
      {wallet ? wallet.balance : "—"}
    </span>
  );
}

export function HealthIndicator() {
  const { t } = useI18n();
  const { data } = useQuery({
    queryKey: ["health"],
    queryFn: () => healthApi.check("/health"),
    refetchInterval: 30_000,
  });
  const ok = data?.ok ?? false;
  return (
    <span
      className="chip"
      style={{ color: ok ? "var(--accent-mission)" : "var(--accent-danger)" }}
      title={`${t("dash.backendStatus")}: ${ok ? "OK" : "DOWN"}`}
    >
      {ok ? <Signal size={12} aria-hidden /> : <SignalZero size={12} aria-hidden />}
      API
    </span>
  );
}

export function Meter({ value, color }: { value: number; color?: string }) {
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <span className="meter" role="img" aria-label={`${clamped}%`}>
      <span
        style={{ width: `${clamped}%`, background: color ?? "var(--accent-ai)" }}
      />
    </span>
  );
}

