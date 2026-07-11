import { useMemo, useState } from "react";
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
        <TacticalButton to="/app/missions/new" variant="primary">
          <Plus size={15} aria-hidden />
          {t("dash.newMission")}
        </TacticalButton>
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
              <TacticalButton to="/app/missions/new" variant="secondary">
                <Plus size={14} aria-hidden />
                {t("dash.newMission")}
              </TacticalButton>
            }
          />
        )}
        {filtered.map((mission) => (
          <MissionDossierCard
            key={mission.id}
            selected={selected?.id === mission.id}
            tone={missionTone(mission.status)}
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

// Case-file accent, driven by mission status — not decoration. Mirrors the
// semantic contract used by badges.tsx's statusChip map: cyan = AI still
// working the case (generating/ready), green = active/completed (safe,
// on-track), red = failed (danger). Gold is deliberately never used here —
// it is reserved for wallet/reward value, not mission state.
function missionTone(status: Mission["status"]): "green" | "cyan" | "red" {
  if (status === "failed") return "red";
  if (status === "generating" || status === "ready") return "cyan";
  return "green";
}

function missionProgress(mission: Mission) {
  if (mission.status === "completed") return 100;
  if (mission.status === "active") return 42;
  if (mission.status === "ready") return 23;
  if (mission.status === "generating") return 8;
  return 0;
}
