import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Plus, Rocket, Wallet, Map, Search, Users } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi, profileApi, walletApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { MissionRow } from "./MissionCard";
import { MissionStatusBadge } from "@/shared/ui/badges";
import type { TranslationKey } from "@/shared/i18n/en";

export function DashboardPage() {
  const { t } = useI18n();
  const missions = useQuery({ queryKey: ["missions"], queryFn: missionsApi.list });
  const wallet = useQuery({ queryKey: ["wallet"], queryFn: walletApi.get });
  const profile = useQuery({ queryKey: ["profile"], queryFn: profileApi.get });

  const active = missions.data?.find(
    (m) => m.status === "active" || m.status === "ready" || m.status === "generating",
  );
  const recent = (missions.data ?? []).slice(0, 5);

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("dash.title")}</h1>
        <Link className="btn btn-primary" to="/app/missions/new">
          <Plus size={15} aria-hidden />
          {t("dash.newMission")}
        </Link>
      </header>

      <div className="dash-grid">
        <div className="panel stat-block col-4 col-half-sm">
          <span className="label">{t("dash.walletBalance")}</span>
          <span className="value" style={{ color: "var(--accent-wallet)" }}>
            {wallet.data ? wallet.data.balance : "—"}
          </span>
        </div>
        <div className="panel stat-block col-4 col-half-sm">
          <span className="label">{t("dash.level")}</span>
          <span className="value">{profile.data ? profile.data.level : "—"}</span>
        </div>
        <div className="panel stat-block col-4 col-half-sm">
          <span className="label">{t("dash.rank")}</span>
          <span className="value" style={{ fontSize: 17 }}>
            {profile.data?.rank ?? "—"}
          </span>
        </div>

        <section className="panel col-8" aria-label={t("dash.activeMission")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("dash.activeMission")}
          </div>
          {missions.isPending && <SkeletonRows rows={3} />}
          {missions.isError && (
            <ErrorState error={missions.error} onRetry={() => missions.refetch()} />
          )}
          {missions.isSuccess && !active && (
            <EmptyState
              title={t("dash.noActiveMission")}
              body={t("dash.noActiveMission.body")}
              action={
                <Link className="btn btn-secondary" to="/app/missions/new">
                  <Plus size={14} aria-hidden />
                  {t("dash.newMission")}
                </Link>
              }
            />
          )}
          {active && (
            <div style={{ padding: 16 }} className="stack">
              <div className="spread">
                <h2>{active.title || t(`type.${active.type}` as TranslationKey)}</h2>
                <MissionStatusBadge status={active.status} />
              </div>
              <p className="muted" style={{ maxWidth: "70ch" }}>
                {active.summary}
              </p>
              <div className="row" style={{ flexWrap: "wrap" }}>
                <Link className="btn btn-primary" to={`/app/missions/${active.id}`}>
                  <Rocket size={14} aria-hidden />
                  {t("dash.enterMission")}
                </Link>
                <Link
                  className="btn btn-secondary"
                  to={`/app/missions/${active.id}/map`}
                >
                  <Map size={14} aria-hidden />
                  {t("nav.map")}
                </Link>
                <Link
                  className="btn btn-secondary"
                  to={`/app/missions/${active.id}/clues`}
                >
                  <Search size={14} aria-hidden />
                  {t("nav.clues")}
                </Link>
                <Link
                  className="btn btn-secondary"
                  to={`/app/missions/${active.id}/characters`}
                >
                  <Users size={14} aria-hidden />
                  {t("nav.characters")}
                </Link>
              </div>
            </div>
          )}
        </section>

        <section className="panel col-4" aria-label={t("dash.quickActions")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("dash.quickActions")}
          </div>
          <div className="item-list">
            <Link className="item-row" to="/app/wallet">
              <Wallet size={15} aria-hidden />
              <span className="grow title">{t("nav.wallet")}</span>
            </Link>
            <Link className="item-row" to="/app/missions">
              <Rocket size={15} aria-hidden />
              <span className="grow title">{t("nav.missions")}</span>
            </Link>
          </div>
        </section>

        <section className="panel col-12" aria-label={t("dash.recentMissions")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("dash.recentMissions")}
          </div>
          {missions.isPending && <SkeletonRows rows={4} />}
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
    </div>
  );
}
