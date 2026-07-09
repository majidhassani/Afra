import { CheckCircle2, Circle, Lock } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { ObjectiveProgress } from "@/shared/ui/game";
import type { Stage } from "@/shared/types/api";

/**
 * The stage progress panel: every mission stage with its status, derived
 * progress, and the concrete requirement lines ("Find 5 clues — 2/5").
 */
export function StageTracker({ stages }: { stages: Stage[] }) {
  const { t } = useI18n();
  if (!stages || stages.length === 0) return null;
  return (
    <section className="stage-tracker" aria-label={t("game.stages")}>
      <h3 className="stage-tracker-title">{t("game.stages")}</h3>
      <ol className="stage-list">
        {stages.map((stage, i) => (
          <li key={stage.id} className={`stage-item ${stage.status}`}>
            <span className="stage-icon" aria-hidden>
              {stage.status === "completed" ? (
                <CheckCircle2 size={17} />
              ) : stage.status === "active" ? (
                <Circle size={17} />
              ) : (
                <Lock size={15} />
              )}
            </span>
            <div className="stage-body">
              <div className="stage-head">
                <span className="stage-num">{i + 1}</span>
                <span className="stage-name">{stage.title}</span>
                {stage.status === "active" && (
                  <span className="stage-pct">{stage.progress}%</span>
                )}
              </div>
              {stage.status === "active" && (
                <>
                  <p className="stage-desc">{stage.description}</p>
                  <ObjectiveProgress value={stage.progress} />
                  <ul className="stage-reqs">
                    {stage.required_actions.map((a, j) => (
                      <li key={j} className={a.done >= a.count ? "done" : ""}>
                        {requirementLabel(t, a.type, a.report, a.count)}
                        <span className="stage-req-count">
                          {a.done}/{a.count}
                        </span>
                      </li>
                    ))}
                  </ul>
                  {(stage.reward.xp > 0 || stage.reward.coins > 0) && (
                    <div className="stage-reward">
                      {t("game.nextReward")}:{" "}
                      {stage.reward.xp > 0 &&
                        t("game.rewardXp", { xp: stage.reward.xp })}{" "}
                      {stage.reward.coins > 0 &&
                        t("game.rewardCoins", { coins: stage.reward.coins })}
                    </div>
                  )}
                </>
              )}
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}

function requirementLabel(
  t: ReturnType<typeof useI18n>["t"],
  type: string,
  report: string | undefined,
  count: number,
): string {
  switch (type) {
    case "visit_locations":
      return t("game.req.visit_locations", { count });
    case "find_clues":
      return t("game.req.find_clues", { count });
    case "confirm_evidence":
      return t("game.req.confirm_evidence", { count });
    case "interview_characters":
      return t("game.req.interview_characters", { count });
    case "submit_report":
      switch (report) {
        case "clue_report":
          return t("report.type.clue_report");
        case "suspect_report":
          return t("report.type.suspect_report");
        case "progress_report":
          return t("report.type.progress_report");
        case "incident_report":
          return t("report.type.incident_report");
        default:
          return t("report.type.final_report");
      }
    case "final_decision":
      return t("game.req.final_decision");
    default:
      return type;
  }
}
