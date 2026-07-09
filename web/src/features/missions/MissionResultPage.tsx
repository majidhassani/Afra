import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  Star,
  Trophy,
  CircleX,
  CheckCircle2,
  Search,
  Coins,
  Sparkles,
  History,
  Home,
} from "lucide-react";
import { useI18n, type TranslationKey } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { ErrorState, SkeletonRows } from "@/shared/ui/states";
import { StageTracker } from "@/features/game/StageTracker";
import { useGameplayStatus } from "@/features/game/useGameplayStatus";
import type { MissionResult } from "@/shared/types/api";

/** prefers-reduced-motion check so count-ups/reveals respect accessibility. */
function reducedMotion(): boolean {
  return (
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

/** Animate a number from 0 to target with an ease-out; instant if reduced. */
function useCountUp(target: number, durationMs = 900): number {
  const [value, setValue] = useState(reducedMotion() ? target : 0);
  const raf = useRef<number | null>(null);
  useEffect(() => {
    if (reducedMotion()) {
      setValue(target);
      return;
    }
    const start = performance.now();
    const tick = (now: number) => {
      const p = Math.min(1, (now - start) / durationMs);
      const eased = 1 - Math.pow(1 - p, 3);
      setValue(Math.round(target * eased));
      if (p < 1) raf.current = requestAnimationFrame(tick);
    };
    raf.current = requestAnimationFrame(tick);
    return () => {
      if (raf.current) cancelAnimationFrame(raf.current);
    };
  }, [target, durationMs]);
  return value;
}

/**
 * MissionResultPage — the standalone, reviewable end-of-mission report. Shows a
 * cinematic win/loss banner, staged star reveal, XP/coin count-ups, per-
 * objective and clue breakdown, and decisions. Reachable after the modal closes
 * and from mission history.
 */
export function MissionResultPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();

  const query = useQuery({
    queryKey: ["mission", missionId, "result"],
    queryFn: () => missionsApi.result(missionId!),
    enabled: !!missionId,
  });
  // Debrief stage recap: how the level was actually played, stage by stage.
  const gameplay = useGameplayStatus(missionId);

  if (query.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={8} />
      </div>
    );
  }
  if (query.isError) {
    return (
      <div className="page">
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
        <div style={{ marginTop: 16 }}>
          <Link to={`/app/missions/${missionId}`} className="btn btn-secondary">
            {t("mission.result.backToMission")}
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="page result-page">
      <ResultReport result={query.data.result} success={query.data.result.success} />
      {gameplay.data && gameplay.data.stages.length > 0 && (
        <StageTracker stages={gameplay.data.stages} />
      )}
    </div>
  );
}

function ResultReport({
  result,
  success,
}: {
  result: MissionResult;
  success: boolean;
}) {
  const { t } = useI18n();
  const [starsShown, setStarsShown] = useState(reducedMotion() ? result.stars : 0);
  const xp = useCountUp(result.xp_reward);
  const coins = useCountUp(result.coin_reward);

  // Reveal stars one at a time.
  useEffect(() => {
    if (reducedMotion()) return;
    setStarsShown(0);
    let n = 0;
    const id = setInterval(() => {
      n += 1;
      setStarsShown(n);
      if (n >= result.stars) clearInterval(id);
    }, 260);
    return () => clearInterval(id);
  }, [result.stars]);

  return (
    <div className={`result-report ${success ? "win" : "loss"}`}>
      <div className="result-banner">
        <div className="result-crest">
          {success ? <Trophy size={34} aria-hidden /> : <CircleX size={34} aria-hidden />}
        </div>
        <div className="result-verdict">
          {result.result_title ||
            (success ? t("mission.result.success") : t("mission.result.failed"))}
        </div>
        <div className="result-stars" aria-label={`${result.stars} / 5`}>
          {[0, 1, 2, 3, 4].map((i) => (
            <Star
              key={i}
              size={26}
              aria-hidden
              className={i < starsShown ? "on star-pop" : ""}
              fill={i < starsShown ? "currentColor" : "none"}
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
            <Sparkles size={14} aria-hidden />
            +{xp} XP
          </span>
          <span className="reward-chip coin">
            <Coins size={14} aria-hidden />
            +{coins}
          </span>
        </div>
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

      <div className="row" style={{ gap: 10, flexWrap: "wrap", marginTop: 8 }}>
        <Link to="/app/dashboard" className="game-btn game-btn-primary">
          <Home size={15} aria-hidden />
          {t("mission.result.backToHub")}
        </Link>
        <Link to="/app/history" className="game-btn game-btn-ghost">
          <History size={15} aria-hidden />
          {t("mission.result.viewHistory")}
        </Link>
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
