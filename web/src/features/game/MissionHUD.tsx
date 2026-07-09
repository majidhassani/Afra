import { Link } from "react-router-dom";
import { Clock3, Search, FileText, Gift, UserRound } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import type { GameplayStatus } from "@/shared/types/api";

/**
 * The always-visible mission HUD strip: current stage, progress, clue goal,
 * suspect status, mission time, next reward, and the report CTA (glowing when
 * the active stage needs a report). Rendered by the AppShell on every
 * mission screen so the player never loses the game state.
 */
export function MissionHUD({
  missionId,
  status,
}: {
  missionId: string;
  status: GameplayStatus;
}) {
  const { t } = useI18n();
  const stage = status.current_stage;
  const progress = status.mission_progress ?? 0;
  const identified = status.suspect_status === "identified";

  return (
    <div className="mission-hud" role="status" aria-label="Mission HUD">
      <div className="hud-cluster hud-stage">
        {stage ? (
          <>
            <span className="hud-kicker">
              {t("game.stageOf", {
                index: status.stage_index,
                count: status.stage_count,
              })}
            </span>
            <span className="hud-stage-title">{stage.title}</span>
          </>
        ) : (
          <span className="hud-stage-title">{t("game.completed")}</span>
        )}
      </div>

      <div className="hud-cluster">
        <span className="hud-ring" style={{ ["--p" as string]: `${progress}` }}>
          <span>{progress}%</span>
        </span>
      </div>

      <div className="hud-cluster">
        <Search size={14} aria-hidden />
        <span className="hud-num">
          {t("game.cluesFound", {
            found: status.clues_found,
            goal: status.clue_goal || status.clues_found,
          })}
        </span>
      </div>

      <div className={`hud-cluster hud-suspect${identified ? " on" : ""}`}>
        <UserRound size={14} aria-hidden />
        <span>
          {t("game.suspect")}:{" "}
          {identified
            ? (status.suspect?.name ?? t("game.suspect.identified"))
            : t("game.suspect.hidden")}
        </span>
      </div>

      <div className="hud-cluster">
        <Clock3 size={14} aria-hidden />
        <span className="hud-num">{status.current_time ?? status.mission.current_time}</span>
      </div>

      {status.next_reward && (
        <div className="hud-cluster hud-reward" title={t("game.nextReward")}>
          <Gift size={14} aria-hidden />
          <span>
            {status.next_reward.xp > 0 &&
              t("game.rewardXp", { xp: status.next_reward.xp })}{" "}
            {status.next_reward.coins > 0 &&
              t("game.rewardCoins", { coins: status.next_reward.coins })}
          </span>
        </div>
      )}

      <Link
        to={`/app/missions/${missionId}/report`}
        className={`hud-report-cta${status.report_pending ? " pending" : ""}`}
        title={status.report_pending ? t("game.reportPending") : t("report.title")}
      >
        <FileText size={15} aria-hidden />
        {t("game.reportCta")}
      </Link>
    </div>
  );
}
