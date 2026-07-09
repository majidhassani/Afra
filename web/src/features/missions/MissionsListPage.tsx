import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Clock3, MapPin, Plus, Rocket } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import {
  MissionDossierCard,
  SelectedMissionBar,
  StatusChip,
  TacticalButton,
} from "@/shared/ui/avds";
import { DifficultyBadge, MissionStatusBadge } from "@/shared/ui/badges";
import type { Mission } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

export function MissionsListPage() {
  const { t } = useI18n();
  const missions = useQuery({ queryKey: ["missions"], queryFn: missionsApi.list });
  const [statusFilter, setStatusFilter] = useState<"all" | Mission["status"]>("all");

  const filtered = useMemo(() => {
    const list = missions.data ?? [];
    if (statusFilter === "all") return list;
    return list.filter((mission) => mission.status === statusFilter);
  }, [missions.data, statusFilter]);

  const selected = filtered[0] ?? missions.data?.[0];

  return (
    <div className="page av-mission-select">
      <header className="page-header">
        <div>
          <h1>{t("missions.title")}</h1>
          <p className="subtitle">{t("missions.empty.body")}</p>
        </div>
        <Link className="av-tactical-button primary" to="/app/missions/new">
          <Plus size={15} aria-hidden />
          {t("dash.newMission")}
        </Link>
      </header>

      <div className="av-filter-rail" aria-label="Mission filters">
        {(["all", "active", "ready", "completed", "generating"] as const).map((status) => (
          <button
            key={status}
            type="button"
            aria-pressed={statusFilter === status}
            onClick={() => setStatusFilter(status)}
          >
            {status === "all"
              ? t("missions.title")
              : t(`missions.status.${status}` as TranslationKey)}
          </button>
        ))}
      </div>

      <div className="av-dossier-grid">
        {missions.isPending && <SkeletonRows rows={6} />}
        {missions.isError && (
          <ErrorState error={missions.error} onRetry={() => missions.refetch()} />
        )}
        {missions.isSuccess && missions.data.length === 0 && (
          <EmptyState
            title={t("missions.empty")}
            body={t("missions.empty.body")}
            action={
              <Link className="av-tactical-button secondary" to="/app/missions/new">
                <Plus size={14} aria-hidden />
                {t("dash.newMission")}
              </Link>
            }
          />
        )}
        {filtered.map((mission, index) => (
          <MissionDossierCard
            key={mission.id}
            selected={selected?.id === mission.id}
            tone={index % 5 === 1 ? "gold" : index % 5 === 2 ? "cyan" : index % 5 === 4 ? "red" : "green"}
            title={mission.title || t(`type.${mission.type}` as TranslationKey)}
            meta={t(`type.${mission.type}` as TranslationKey)}
            summary={mission.summary}
            progress={missionProgress(mission)}
            stats={
              <>
                <StatusChip tone="cyan">
                  <Clock3 size={12} aria-hidden />
                  {mission.current_time}
                </StatusChip>
                {mission.region && (
                  <StatusChip>
                    <MapPin size={12} aria-hidden />
                    {mission.region}
                  </StatusChip>
                )}
                <DifficultyBadge difficulty={mission.difficulty} />
                <MissionStatusBadge status={mission.status} />
              </>
            }
            action={
              <TacticalButton
                to={`/app/missions/${mission.id}`}
                variant={mission.status === "active" || mission.status === "ready" ? "primary" : "ghost"}
              >
                <Rocket size={14} aria-hidden />
                {mission.status === "active" || mission.status === "ready"
                  ? t("hub.resume")
                  : t("missions.title")}
              </TacticalButton>
            }
          />
        ))}
      </div>

      {selected && (
        <SelectedMissionBar
          title={selected.title || t(`type.${selected.type}` as TranslationKey)}
          meta={selected.summary}
          action={
            <TacticalButton to={`/app/missions/${selected.id}`}>
              <Rocket size={15} aria-hidden />
              {t("dash.enterMission")}
            </TacticalButton>
          }
        />
      )}
    </div>
  );
}

function missionProgress(mission: Mission) {
  if (mission.status === "completed") return 100;
  if (mission.status === "active") return 42;
  if (mission.status === "ready") return 23;
  if (mission.status === "generating") return 8;
  return 0;
}
