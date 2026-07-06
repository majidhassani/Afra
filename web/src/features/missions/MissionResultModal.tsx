import { Link } from "react-router-dom";
import {
  Star,
  Trophy,
  CircleX,
  CheckCircle2,
  Search,
  Coins,
  Sparkles,
  X,
} from "lucide-react";
import { useI18n, type TranslationKey } from "@/shared/i18n";
import type { MissionResult } from "@/shared/types/api";

/**
 * MissionResultModal — the satisfying end-of-mission reveal. Shows a clear
 * win/loss verdict, star rating, score, per-objective and clue breakdown, the
 * key decisions, and the XP/coin rewards.
 */
export function MissionResultModal({
  result,
  onClose,
  missionId,
}: {
  result: MissionResult;
  onClose: () => void;
  /** When set, the modal offers a link to the standalone result report. */
  missionId?: string;
}) {
  const { t } = useI18n();
  const success = result.success;

  return (
    <div className="result-overlay" role="dialog" aria-modal="true">
      <div className={`result-card ${success ? "win" : "loss"}`}>
        <button
          className="result-close"
          onClick={onClose}
          aria-label={t("common.close")}
        >
          <X size={18} aria-hidden />
        </button>

        <div className="result-crest">
          {success ? <Trophy size={30} aria-hidden /> : <CircleX size={30} aria-hidden />}
        </div>
        <div className="result-verdict">
          {result.result_title ||
            (success ? t("mission.result.success") : t("mission.result.failed"))}
        </div>

        <div className="result-stars" aria-label={`${result.stars} / 5`}>
          {[0, 1, 2, 3, 4].map((i) => (
            <Star
              key={i}
              size={22}
              aria-hidden
              className={i < result.stars ? "on" : ""}
              fill={i < result.stars ? "currentColor" : "none"}
            />
          ))}
        </div>

        <div className="result-score">
          <span className="mono-num">{result.score}</span>
          <span className="faint">{t("mission.result.score")}</span>
        </div>

        {result.result_summary && (
          <p className="result-summary" style={{ unicodeBidi: "plaintext" }}>
            {result.result_summary}
          </p>
        )}

        <div className="result-rewards">
          <span className="reward-chip xp">
            <Sparkles size={13} aria-hidden />
            +{result.xp_reward} XP
          </span>
          <span className="reward-chip coin">
            <Coins size={13} aria-hidden />
            +{result.coin_reward}
          </span>
        </div>

        <div className="result-lists">
          <ResultList
            titleKey="mission.result.objectivesDone"
            items={result.completed_objectives}
            icon={CheckCircle2}
            tone="good"
          />
          <ResultList
            titleKey="mission.result.objectivesFailed"
            items={result.failed_objectives}
            icon={CircleX}
            tone="bad"
          />
          <ResultList
            titleKey="mission.result.cluesFound"
            items={result.critical_clues_found}
            icon={Search}
            tone="good"
          />
          <ResultList
            titleKey="mission.result.cluesMissed"
            items={result.critical_clues_missed}
            icon={Search}
            tone="bad"
          />
          <ResultList
            titleKey="mission.result.goodDecisions"
            items={result.good_decisions}
            icon={CheckCircle2}
            tone="good"
          />
          <ResultList
            titleKey="mission.result.badDecisions"
            items={result.bad_decisions}
            icon={CircleX}
            tone="bad"
          />
        </div>

        <div className="row result-cta" style={{ gap: 10, flexWrap: "wrap", justifyContent: "center" }}>
          {missionId && (
            <Link
              to={`/app/missions/${missionId}/result`}
              className="game-btn game-btn-ghost"
              onClick={onClose}
            >
              {t("mission.result.viewFullReport")}
            </Link>
          )}
          <Link to="/app/dashboard" className="game-btn game-btn-primary">
            {t("mission.result.backToHub")}
          </Link>
        </div>
      </div>
    </div>
  );
}

function ResultList({
  titleKey,
  items,
  icon: Icon,
  tone,
}: {
  titleKey: TranslationKey;
  items: string[] | null | undefined;
  icon: typeof CheckCircle2;
  tone: "good" | "bad";
}) {
  const { t } = useI18n();
  if (!items || items.length === 0) return null;
  return (
    <div className="result-list">
      <div className="band-title">{t(titleKey)}</div>
      {items.map((item) => (
        <div key={item} className={`result-line ${tone}`}>
          <Icon size={13} aria-hidden />
          <span style={{ unicodeBidi: "plaintext" }}>{item}</span>
        </div>
      ))}
    </div>
  );
}
