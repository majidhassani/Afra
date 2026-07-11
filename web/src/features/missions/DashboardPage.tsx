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
  Plus,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi, profileApi, walletApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { MissionRow } from "./MissionCard";
import { MissionStatusBadge } from "@/shared/ui/badges";
import { WalletBalance } from "@/shared/ui/game";
import {
  MissionMapPanel,
  ProgressRing,
  ResourceChip,
  SelectedMissionBar,
  StatusChip,
  TacticalButton,
} from "@/shared/ui/avds";
import { useEnvironmentTheme, environmentFor } from "@/shared/theme/environment";
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
  const activeProgress =
    active?.status === "completed" ? 100 : active?.status === "active" ? 32 : active ? 12 : 0;
  const deployTarget = active ? `/app/missions/${active.id}` : "/app/missions/new";

  // The lobby wears the biome of the active operation.
  useEnvironmentTheme(active ? environmentFor(active) : null);

  return (
    <div className="page av-home-command stack" style={{ gap: 20 }}>
      <section className="av-command-hero" aria-label={t("common.appName")}>
        <div className="av-command-meta">
          <ResourceChip
            icon={<ShieldCheck size={14} aria-hidden />}
            label={t("hub.rank")}
            value={profile.data?.rank ?? "—"}
            tone="green"
          />
          <ResourceChip
            label={t("hub.level")}
            value={profile.data?.level ?? "—"}
            tone="cyan"
          />
          <ResourceChip value={<WalletBalance balance={wallet.data?.balance} />} tone="gold" />
        </div>

        <div className="av-command-identity">
          <span className="av-eyebrow">{t("hub.welcome", { name: agentName || t("hub.agent") })}</span>
          <h1>{t("common.appName")}</h1>
          <p>{t("hub.tagline")}</p>
        </div>

        <Link className="av-deploy-orb" to={deployTarget}>
          <span className="av-deploy-mark" aria-hidden>
            A
          </span>
          <strong>{active ? t("dash.enterMission") : t("dash.newMission")}</strong>
          <em>{active ? t("hub.resume") : t("hub.deploy")}</em>
        </Link>
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
              action={
                <TacticalButton to="/app/missions/new" variant="secondary">
                  <Plus size={14} aria-hidden />
                  {t("dash.newMission")}
                </TacticalButton>
              }
            />
          </div>
        )}
        {active && (
          <SelectedMissionBar
            title={active.title || t(`type.${active.type}` as TranslationKey)}
            meta={
              <span className="row" style={{ gap: 10, flexWrap: "wrap" }}>
                <MissionStatusBadge status={active.status} />
                <StatusChip tone="cyan">{active.region || t("mission.region")}</StatusChip>
                <StatusChip tone="gold">{t(`difficulty.${active.difficulty}` as TranslationKey)}</StatusChip>
              </span>
            }
            action={
              <div className="row" style={{ gap: 10, flexWrap: "wrap" }}>
                <ProgressRing value={activeProgress} />
                <TacticalButton to={`/app/missions/${active.id}`}>
                  <Rocket size={15} aria-hidden />
                  {t("hub.resume")}
                </TacticalButton>
                <TacticalButton to={`/app/missions/${active.id}/map`} variant="ghost">
                  <Map size={15} aria-hidden />
                  {t("nav.map")}
                </TacticalButton>
              </div>
            }
          />
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

      <MissionMapPanel
        title={active?.region || t("nav.map")}
        subtitle={active?.title || t("dash.noActiveMission")}
        to={active ? `/app/missions/${active.id}/map` : undefined}
        linkLabel={t("hud.openMap")}
      />

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
