import { Link, useNavigate, useParams } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Map,
  Users,
  Search,
  NotebookPen,
  Clock3,
  Radio,
  Archive,
  CircleCheck,
  Circle,
  CircleX,
  Loader2,
  MapPin,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import {
  MissionStatusBadge,
  DifficultyBadge,
  AvatarPlaceholder,
} from "@/shared/ui/badges";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import { useMissionDashboard } from "./missionQueries";
import { toast } from "@/shared/ui/toast";
import type { Objective } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

export function MissionDashboardPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const dashboard = useMissionDashboard(missionId);

  const events = useQuery({
    queryKey: ["mission", missionId, "events"],
    queryFn: () => missionsApi.events(missionId!, 8),
    enabled: !!missionId,
  });

  const archive = useMutation({
    mutationFn: () => missionsApi.archive(missionId!),
    onSuccess: () => {
      toast("success", t("missions.archived"));
      void queryClient.invalidateQueries({ queryKey: ["missions"] });
      navigate("/app/missions");
    },
  });

  if (dashboard.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={8} />
      </div>
    );
  }
  if (dashboard.isError) {
    return (
      <div className="page">
        <ErrorState error={dashboard.error} onRetry={() => dashboard.refetch()} />
      </div>
    );
  }

  const { mission, characters, clues, locations } = dashboard.data;
  const objectives: Objective[] = Array.isArray(mission.objectives)
    ? mission.objectives
    : [];
  const publicState =
    mission.public_state && typeof mission.public_state === "object"
      ? Object.entries(mission.public_state)
      : [];

  if (mission.status === "generating") {
    return (
      <div className="page">
        <div className="state-box" style={{ minHeight: "50dvh" }}>
          <Loader2 size={28} className="spin" aria-hidden />
          <div className="state-title">{t("missions.generating")}</div>
          <p className="faint">{t("missions.generating.body")}</p>
          <div className="stack" style={{ gap: 4, alignItems: "center" }}>
            {events.data?.slice(0, 5).map((ev) => (
              <span key={ev.id} className="faint">
                {ev.type.replace(/_/g, " ")}
              </span>
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (mission.status === "failed") {
    return (
      <div className="page">
        <div className="state-box" style={{ minHeight: "50dvh" }}>
          <CircleX size={28} color="var(--accent-danger)" aria-hidden />
          <div className="state-title">{t("missions.generationFailed")}</div>
          <Link className="btn btn-secondary" to="/app/missions/new">
            {t("dash.newMission")}
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>{mission.title}</h1>
          <div className="row subtitle" style={{ flexWrap: "wrap" }}>
            <span>{t(`type.${mission.type}` as TranslationKey)}</span>
            <DifficultyBadge difficulty={mission.difficulty} />
            <MissionStatusBadge status={mission.status} />
            {mission.region && (
              <span className="row faint">
                <MapPin size={12} aria-hidden />
                {mission.region}
              </span>
            )}
            <span className="chip mono-num">
              <Clock3 size={12} aria-hidden />
              {mission.current_time}
            </span>
          </div>
        </div>
        <div className="row">
          <button
            className="btn btn-ghost"
            onClick={() => archive.mutate()}
            disabled={archive.isPending}
            title={t("missions.archive")}
          >
            <Archive size={15} aria-hidden />
            {t("missions.archive")}
          </button>
        </div>
      </header>

      <div className="dash-grid">
        <section className="panel col-8" aria-label={t("mission.briefing")}>
          <div className="band">
            <div className="band-title">{t("mission.briefing")}</div>
            <p style={{ whiteSpace: "pre-wrap", unicodeBidi: "plaintext" }}>
              {mission.briefing}
            </p>
          </div>
          <div className="band" style={{ borderBottom: "none" }}>
            <div className="band-title">{t("mission.objectives")}</div>
            {objectives.length === 0 && (
              <p className="faint">{t("common.empty.title")}</p>
            )}
            <ul className="stack" style={{ listStyle: "none", margin: 0, padding: 0 }}>
              {objectives.map((obj) => (
                <li key={obj.id} className="row" style={{ alignItems: "flex-start" }}>
                  {obj.status === "completed" ? (
                    <CircleCheck size={16} color="var(--accent-mission)" aria-hidden />
                  ) : obj.status === "failed" ? (
                    <CircleX size={16} color="var(--accent-danger)" aria-hidden />
                  ) : (
                    <Circle size={16} color="var(--text-faint)" aria-hidden />
                  )}
                  <div>
                    <div className="row" style={{ flexWrap: "wrap", gap: 6 }}>
                      <strong>{obj.title}</strong>
                      {obj.optional && (
                        <span className="chip">{t("mission.objective.optional")}</span>
                      )}
                      {obj.required_clues > 0 && (
                        <span className="faint">
                          {t("mission.objective.requiredClues", {
                            n: obj.required_clues,
                          })}
                        </span>
                      )}
                    </div>
                    <p className="muted">{obj.description}</p>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </section>

        <div className="col-4 stack">
          {publicState.length > 0 && (
            <section className="panel" aria-label={t("mission.publicState")}>
              <div className="band-title" style={{ padding: "14px 16px 0" }}>
                {t("mission.publicState")}
              </div>
              <div className="item-list">
                {publicState.map(([key, value]) => (
                  <div key={key} className="item-row">
                    <span className="grow sub">{key}</span>
                    <span style={{ fontSize: 13, unicodeBidi: "plaintext" }}>
                      {String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}

          <section className="panel" aria-label={t("guidance.title")}>
            <div style={{ padding: 16 }}>
              <GuidancePanel missionId={mission.id} screen="mission_dashboard" />
            </div>
          </section>
        </div>

        <section className="panel col-4" aria-label={t("mission.locations")}>
          <Link className="spread" style={{ padding: "14px 16px 0" }} to={`/app/missions/${mission.id}/map`}>
            <span className="band-title" style={{ marginBottom: 0 }}>
              {t("mission.locations")} ({locations.length})
            </span>
            <Map size={14} aria-hidden />
          </Link>
          <div className="item-list">
            {locations.slice(0, 4).map((marker) => (
              <Link
                key={marker.id}
                className="item-row"
                to={`/app/missions/${mission.id}/locations/${marker.id}`}
              >
                <MapPin size={14} aria-hidden />
                <span className="grow title">{marker.name}</span>
                {marker.is_locked && <span className="chip">{t("map.locked")}</span>}
              </Link>
            ))}
            {locations.length === 0 && (
              <EmptyState title={t("common.empty.title")} />
            )}
          </div>
        </section>

        <section className="panel col-4" aria-label={t("mission.characters")}>
          <Link className="spread" style={{ padding: "14px 16px 0" }} to={`/app/missions/${mission.id}/characters`}>
            <span className="band-title" style={{ marginBottom: 0 }}>
              {t("mission.characters")} ({characters.length})
            </span>
            <Users size={14} aria-hidden />
          </Link>
          <div className="item-list">
            {characters.slice(0, 4).map((c) => (
              <Link
                key={c.id}
                className="item-row"
                to={`/app/missions/${mission.id}/characters/${c.id}`}
              >
                <AvatarPlaceholder name={c.name} prompt={c.avatar_prompt} />
                <div className="grow">
                  <div className="title">{c.name}</div>
                  <div className="sub">{c.role}</div>
                </div>
              </Link>
            ))}
            {characters.length === 0 && (
              <EmptyState title={t("chars.empty")} />
            )}
          </div>
        </section>

        <section className="panel col-4" aria-label={t("mission.clues")}>
          <Link className="spread" style={{ padding: "14px 16px 0" }} to={`/app/missions/${mission.id}/clues`}>
            <span className="band-title" style={{ marginBottom: 0 }}>
              {t("mission.clues")} ({clues.length})
            </span>
            <Search size={14} aria-hidden />
          </Link>
          <div className="item-list">
            {clues.slice(0, 4).map((clue) => (
              <Link
                key={clue.id}
                className="item-row"
                to={`/app/missions/${mission.id}/clues/${clue.id}`}
              >
                <Search size={14} aria-hidden />
                <div className="grow">
                  <div className="title">{clue.title}</div>
                  <div className="sub">{clue.short_description}</div>
                </div>
              </Link>
            ))}
            {clues.length === 0 && (
              <EmptyState title={t("clues.empty")} />
            )}
          </div>
        </section>

        <section className="panel col-12" aria-label={t("mission.events")}>
          <Link
            className="spread"
            style={{ padding: "14px 16px 0" }}
            to={`/app/missions/${mission.id}/events`}
          >
            <span className="band-title" style={{ marginBottom: 0 }}>
              {t("mission.events")}
            </span>
            <Radio size={14} aria-hidden />
          </Link>
          <div className="item-list">
            {events.data?.slice(0, 5).map((ev) => (
              <div key={ev.id} className="item-row">
                <span className="grow sub">{ev.type.replace(/_/g, " ")}</span>
                <span className="faint mono-num">
                  {new Date(ev.created_at).toLocaleTimeString()}
                </span>
              </div>
            ))}
            {events.isSuccess && events.data.length === 0 && (
              <EmptyState title={t("events.empty")} />
            )}
          </div>
        </section>

        <section className="panel col-12" aria-label={t("nav.journal")}>
          <div className="item-list">
            <Link className="item-row" to={`/app/missions/${mission.id}/journal`}>
              <NotebookPen size={15} aria-hidden />
              <span className="grow title">{t("nav.journal")}</span>
            </Link>
            <Link className="item-row" to={`/app/missions/${mission.id}/time`}>
              <Clock3 size={15} aria-hidden />
              <span className="grow title">{t("time.title")}</span>
              <span className="chip mono-num">{mission.current_time}</span>
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
}
