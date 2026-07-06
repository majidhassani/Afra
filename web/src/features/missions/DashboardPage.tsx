import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Rocket,
  Map,
  Sparkles,
  PawPrint,
  Search,
  Waves,
  Compass,
  Mountain,
  Handshake,
  Stethoscope,
  ShieldCheck,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi, profileApi, walletApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { MissionRow } from "./MissionCard";
import { MissionStatusBadge } from "@/shared/ui/badges";
import { GameButton, WalletBalance } from "@/shared/ui/game";
import type { MissionType } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

const missionTypes: { type: MissionType; icon: typeof PawPrint }[] = [
  { type: "wildlife_rescue", icon: PawPrint },
  { type: "detective", icon: Search },
  { type: "disaster_response", icon: Waves },
  { type: "exploration", icon: Compass },
  { type: "survival", icon: Mountain },
  { type: "diplomacy", icon: Handshake },
  { type: "medical_mystery", icon: Stethoscope },
];

export function DashboardPage() {
  const { t } = useI18n();
  const missions = useQuery({ queryKey: ["missions"], queryFn: missionsApi.list });
  const wallet = useQuery({ queryKey: ["wallet"], queryFn: walletApi.get });
  const profile = useQuery({ queryKey: ["profile"], queryFn: profileApi.get });

  const active = missions.data?.find(
    (m) => m.status === "active" || m.status === "ready" || m.status === "generating",
  );
  const recent = (missions.data ?? []).slice(0, 5);
  const agentName = profile.data?.display_name ?? "";

  return (
    <div className="page stack" style={{ gap: 18 }}>
      {/* Hero */}
      <section className="hub-hero">
        <div className="spread" style={{ alignItems: "flex-start", flexWrap: "wrap" }}>
          <div className="stack" style={{ gap: 8, minWidth: 0 }}>
            <span className="eyebrow row" style={{ gap: 6 }}>
              <ShieldCheck size={13} aria-hidden />
              {t("hub.agent")}
            </span>
            <h1>{t("hub.welcome", { name: agentName })}</h1>
            <p className="muted" style={{ maxWidth: "56ch" }}>
              {t("hub.tagline")}
            </p>
          </div>
          <div className="stack" style={{ gap: 10, alignItems: "flex-end" }}>
            <WalletBalance balance={wallet.data?.balance} />
            <div className="row" style={{ gap: 6 }}>
              <span className="status-chip cat-guide">
                {t("hub.rank")}: {profile.data?.rank ?? "—"}
              </span>
              <span className="status-chip">
                {t("hub.level")} {profile.data?.level ?? "—"}
              </span>
            </div>
          </div>
        </div>
      </section>

      {/* Active operation */}
      <section aria-label={t("hub.activeMission")}>
        {missions.isPending && (
          <div className="panel">
            <SkeletonRows rows={3} />
          </div>
        )}
        {missions.isError && (
          <div className="panel">
            <ErrorState error={missions.error} onRetry={() => missions.refetch()} />
          </div>
        )}
        {missions.isSuccess && !active && (
          <div className="panel">
            <EmptyState
              title={t("dash.noActiveMission")}
              body={t("dash.noActiveMission.body")}
            />
          </div>
        )}
        {active && (
          <div className="tac-card" style={{ padding: 20 }}>
            <div className="spread" style={{ flexWrap: "wrap", gap: 12 }}>
              <div className="stack" style={{ gap: 6, minWidth: 0 }}>
                <div className="row" style={{ gap: 8, flexWrap: "wrap" }}>
                  <span className="eyebrow">{t("hub.activeMission")}</span>
                  <MissionStatusBadge status={active.status} />
                </div>
                <h2 style={{ fontSize: 20 }}>
                  {active.title || t(`type.${active.type}` as TranslationKey)}
                </h2>
                <p className="muted" style={{ maxWidth: "64ch" }}>
                  {active.summary}
                </p>
              </div>
            </div>
            <div className="row" style={{ marginTop: 16, flexWrap: "wrap", gap: 10 }}>
              <Link to={`/app/missions/${active.id}`}>
                <GameButton variant="primary">
                  <Rocket size={15} aria-hidden />
                  {t("hub.resume")}
                </GameButton>
              </Link>
              <Link to={`/app/missions/${active.id}/map`}>
                <GameButton variant="ghost">
                  <Map size={15} aria-hidden />
                  {t("nav.map")}
                </GameButton>
              </Link>
            </div>
          </div>
        )}
      </section>

      {/* Mission type tiles */}
      <section aria-label={t("hub.chooseType")}>
        <div className="band-title">{t("hub.chooseType")}</div>
        <div className="type-grid">
          {missionTypes.map(({ type, icon: Icon }) => (
            <Link key={type} className="type-tile" to={`/app/missions/new?type=${type}`}>
              <span className="type-icon">
                <Icon size={20} aria-hidden />
              </span>
              <span className="type-name">{t(`type.${type}` as TranslationKey)}</span>
              <span className="faint row" style={{ gap: 5 }}>
                <Sparkles size={11} aria-hidden />
                {t("hub.deploy")}
              </span>
            </Link>
          ))}
        </div>
      </section>

      {/* Recent operations */}
      <section className="panel" aria-label={t("hub.recent")}>
        <div className="band-title" style={{ padding: "14px 16px 0" }}>
          {t("hub.recent")}
        </div>
        {missions.isPending && <SkeletonRows rows={3} />}
        {missions.isSuccess && recent.length === 0 && (
          <EmptyState title={t("missions.empty")} body={t("missions.empty.body")} />
        )}
        <div className="item-list">
          {recent.map((m) => (
            <MissionRow key={m.id} mission={m} />
          ))}
        </div>
      </section>
    </div>
  );
}
