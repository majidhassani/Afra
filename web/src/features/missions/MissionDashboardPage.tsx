import { useState } from "react";
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
  CircleDot,
  MapPin,
  Target,
  Flag,
  Navigation,
  Coins,
  Trophy,
  AlertTriangle,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import { MissionStatusBadge, DifficultyBadge } from "@/shared/ui/badges";
import { Avatar } from "@/shared/ui/Avatar";
import {
  GameButton,
  RiskMeter,
  ObjectiveProgress,
  HudStat,
  LoadingScreen,
  WalletBalance,
} from "@/shared/ui/game";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import { TimelineLog } from "./TimelineLog";
import { NextActionCard } from "./NextActionCard";
import { MissionResultModal } from "./MissionResultModal";
import { useMissionDashboard } from "./missionQueries";
import { deriveHud } from "./hud";
import { toast } from "@/shared/ui/toast";
import type { MissionResult, Objective } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

export function MissionDashboardPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const dashboard = useMissionDashboard(missionId);
  const [resultModal, setResultModal] = useState<MissionResult | null>(null);

  const events = useQuery({
    queryKey: ["mission", missionId, "events"],
    queryFn: () => missionsApi.events(missionId!, 8),
    enabled: !!missionId,
  });

  const timeline = useQuery({
    queryKey: ["mission", missionId, "timeline"],
    queryFn: () => missionsApi.timeline(missionId!),
    enabled: !!missionId,
  });

  // Record the "ready for final decision" timeline milestone (server dedupes).
  const canComplete = dashboard.data?.can_complete === true;
  useQuery({
    queryKey: ["mission", missionId, "ready-milestone"],
    queryFn: () => missionsApi.completionCheck(missionId!),
    enabled: !!missionId && canComplete,
    staleTime: Infinity,
  });

  const archive = useMutation({
    mutationFn: () => missionsApi.archive(missionId!),
    onSuccess: () => {
      toast("success", t("missions.archived"));
      void queryClient.invalidateQueries({ queryKey: ["missions"] });
      navigate("/app/missions");
    },
  });

  const complete = useMutation({
    mutationFn: () =>
      missionsApi.complete(missionId!, {
        outcome: "Submit final mission judgment",
        reasoning: "Player requested completion from the mission command HUD.",
      }),
    onSuccess: (res) => {
      // A ready mission returns the judged result; an unready one returns a
      // completion check (can_complete=false) which the readiness panel covers.
      if (res && "result_title" in res && res.can_complete !== false) {
        setResultModal(res as MissionResult);
      } else {
        toast("success", t("mission.result.ready"));
      }
      void queryClient.invalidateQueries({ queryKey: ["mission", missionId] });
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "events"],
      });
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

  const data = dashboard.data;
  const { mission, characters, clues, locations } = data;
  const objectives: Objective[] =
    data.objectives && data.objectives.length > 0
      ? [...data.objectives, ...(data.completed_objectives ?? [])]
      : Array.isArray(mission.objectives)
        ? mission.objectives
        : [];
  const hud = deriveHud(mission, objectives, clues, locations);
  const primaryObjective = data.primary_objective ?? hud.primary;
  const progress = data.mission_progress ?? hud.progress;
  const risk = data.risk_score ?? hud.risk;
  const riskBand = risk >= 66 ? "high" : risk >= 33 ? "med" : "low";
  const winConditions = data.win_conditions ?? hud.winConditions;
  const failureConditions = data.failure_conditions ?? [];
  const recommendedAction = data.next_recommended_actions?.[0];
  const recommendedLocation =
    (recommendedAction?.target_type === "location" &&
      locations.find((l) => l.id === recommendedAction.target_id)) ||
    hud.recommended;
  const riskLabel =
    riskBand === "high"
      ? t("hud.riskHigh")
      : riskBand === "med"
        ? t("hud.riskMed")
        : t("hud.riskLow");

  if (mission.status === "generating") {
    return (
      <div className="page">
        <LoadingScreen
          title={t("missions.generating")}
          steps={[
            t("loading.mission.world"),
            t("loading.mission.characters"),
            t("loading.mission.clues"),
            t("loading.mission.map"),
          ]}
          activeStep={Math.min(3, events.data?.length ?? 0)}
        />
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
        <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
          <Link to={`/app/missions/${mission.id}/map`}>
            <GameButton variant="primary">
              <Map size={15} aria-hidden />
              {t("hud.openMap")}
            </GameButton>
          </Link>
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

      {/* Command-center HUD */}
      <div className="hud" style={{ marginBottom: 14 }}>
        <HudStat label={t("hud.objective")}>
          <div style={{ fontSize: 14, fontWeight: 600 }}>
            {primaryObjective ? primaryObjective.title : t("hud.noObjective")}
          </div>
        </HudStat>
        <HudStat label={t("hud.progress")}>
          <div className="stack" style={{ gap: 6 }}>
            <span className="mono-num">{progress}%</span>
            <ObjectiveProgress value={progress} />
          </div>
        </HudStat>
        <HudStat label={t("hud.risk")}>
          <div className="stack" style={{ gap: 6 }}>
            <span
              className="mono-num"
              style={{
                color:
                  riskBand === "high"
                    ? "var(--accent-danger)"
                    : riskBand === "med"
                      ? "var(--accent-wallet)"
                      : "var(--accent-mission)",
              }}
            >
              {riskLabel} · {risk}
            </span>
            <RiskMeter value={risk} />
          </div>
        </HudStat>
        <HudStat label={t("hud.timeRemaining")}>
          <span className="mono-num row" style={{ gap: 6 }}>
            <Clock3 size={14} aria-hidden />
            {data.time_remaining || mission.current_time}
          </span>
        </HudStat>
        <HudStat label={t("dash.walletBalance")}>
          <span className="row" style={{ gap: 6 }}>
            <Coins size={14} aria-hidden />
            <WalletBalance balance={data.wallet_balance} />
          </span>
        </HudStat>
      </div>

      {/* Recommended next move — what to do and why it matters */}
      {recommendedAction ? (
        <div style={{ marginBottom: 14 }}>
          <NextActionCard action={recommendedAction} missionId={mission.id} />
        </div>
      ) : recommendedLocation ? (
        <Link
          to={`/app/missions/${mission.id}/locations/${recommendedLocation.id}`}
          className="reco-banner"
          style={{ marginBottom: 14 }}
        >
          <Navigation size={18} className="reco-icon" aria-hidden />
          <div className="grow">
            <div className="faint" style={{ fontSize: 11 }}>
              {t("hud.recommended")}
            </div>
            <strong>{recommendedLocation.name}</strong>
          </div>
          <span className="status-chip cat-guide">{t("map.marker.recommended")}</span>
        </Link>
      ) : null}

      {mission.status === "completed" && (
        <section className="mission-result-panel" aria-label={t("mission.result.title")}>
          <Trophy size={24} aria-hidden />
          <div className="grow">
            <div className="band-title">{t("mission.result.title")}</div>
            <p className="muted">{t("mission.result.completed")}</p>
          </div>
          {data.result != null && typeof data.result === "object" ? (
            <GameButton
              variant="mission"
              size="sm"
              onClick={() => setResultModal(data.result as MissionResult)}
            >
              {t("mission.result.viewReport")}
            </GameButton>
          ) : null}
        </section>
      )}

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
            <div className="stack" style={{ gap: 0 }}>
              {objectives.map((obj) => (
                <ObjectiveItem key={obj.id} obj={obj} />
              ))}
            </div>
          </div>
        </section>

        <div className="col-4 stack">
          {winConditions.length > 0 && (
            <section className="panel" aria-label={t("hud.winConditions")}>
              <div className="band-title" style={{ padding: "14px 16px 0" }}>
                {t("hud.winConditions")}
              </div>
              <div className="item-list">
                {winConditions.map((cond) => (
                  <div key={cond} className="item-row" style={{ gap: 10 }}>
                    <Flag size={14} color="var(--accent-mission)" aria-hidden />
                    <span className="grow" style={{ unicodeBidi: "plaintext" }}>
                      {cond}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}

          <section className="panel" aria-label={t("mission.finish.title")}>
            <div className="band" style={{ borderBottom: "none" }}>
              <div className="band-title">{t("mission.finish.title")}</div>
              {data.can_complete ? (
                <p className="muted">{t("mission.finish.ready")}</p>
              ) : (
                <div className="stack" style={{ gap: 8 }}>
                  <p className="muted">{t("mission.finish.notReady")}</p>
                  {(data.missing_requirements ?? []).slice(0, 3).map((req) => (
                    <div key={req} className="item-row" style={{ padding: 0 }}>
                      <AlertTriangle
                        size={14}
                        color="var(--accent-wallet)"
                        aria-hidden
                      />
                      <span className="sub">{req}</span>
                    </div>
                  ))}
                </div>
              )}
              <div style={{ marginTop: 12 }}>
                <GameButton
                  variant={data.can_complete ? "mission" : "ghost"}
                  disabled={!data.can_complete || complete.isPending}
                  onClick={() => complete.mutate()}
                >
                  <Trophy size={15} aria-hidden />
                  {t("mission.finish.cta")}
                </GameButton>
              </div>
            </div>
          </section>

          {failureConditions.length > 0 && (
            <section className="panel" aria-label={t("hud.failureConditions")}>
              <div className="band-title" style={{ padding: "14px 16px 0" }}>
                {t("hud.failureConditions")}
              </div>
              <div className="item-list">
                {failureConditions.map((cond) => (
                  <div key={cond} className="item-row" style={{ gap: 10 }}>
                    <AlertTriangle
                      size={14}
                      color="var(--accent-danger)"
                      aria-hidden
                    />
                    <span className="grow" style={{ unicodeBidi: "plaintext" }}>
                      {cond}
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
                <Avatar name={c.name} category={c.category} size="sm" />
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

        <section className="panel col-12" aria-label={t("timeline.title")}>
          <Link
            className="spread"
            style={{ padding: "14px 16px 4px" }}
            to={`/app/missions/${mission.id}/timeline`}
          >
            <span className="band-title" style={{ marginBottom: 0 }}>
              {t("timeline.title")}
            </span>
            <Radio size={14} aria-hidden />
          </Link>
          <div style={{ padding: "8px 16px 16px" }}>
            <TimelineLog
              items={timeline.data?.items}
              missionId={mission.id}
              limit={6}
            />
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

      {resultModal && (
        <MissionResultModal
          result={resultModal}
          onClose={() => setResultModal(null)}
        />
      )}
    </div>
  );
}

function ObjectiveItem({ obj }: { obj: Objective }) {
  const { t } = useI18n();
  const isPrimary = obj.type === "primary";
  const isFinal = obj.type === "final";
  const optional = obj.optional || obj.type === "optional";
  const progress = typeof obj.progress === "number" ? obj.progress : undefined;

  const Icon =
    obj.status === "completed"
      ? CircleCheck
      : obj.status === "failed"
        ? CircleX
        : obj.status === "locked"
          ? Circle
          : isPrimary
            ? Target
            : CircleDot;
  const iconColor =
    obj.status === "completed"
      ? "var(--accent-mission)"
      : obj.status === "failed"
        ? "var(--accent-danger)"
        : isPrimary
          ? "var(--accent-ai)"
          : "var(--text-faint)";

  return (
    <div className={`objective-row${obj.status === "completed" ? " completed" : ""}`}>
      <Icon size={17} color={iconColor} aria-hidden style={{ marginTop: 2, flexShrink: 0 }} />
      <div className="grow" style={{ minWidth: 0 }}>
        <div className="row" style={{ flexWrap: "wrap", gap: 6 }}>
          <strong className="obj-title">{obj.title}</strong>
          {isPrimary && <span className="status-chip cat-guide">{t("hud.objective")}</span>}
          {isFinal && <span className="status-chip cat-field">{obj.type}</span>}
          {optional && (
            <span className="status-chip">{t("mission.objective.optional")}</span>
          )}
        </div>
        <p className="muted" style={{ marginTop: 4 }}>
          {obj.description}
        </p>
        {progress !== undefined && obj.status !== "locked" && (
          <div className="row" style={{ gap: 8, marginTop: 8 }}>
            <ObjectiveProgress value={progress} />
            <span className="faint mono-num">{progress}%</span>
          </div>
        )}
      </div>
    </div>
  );
}
