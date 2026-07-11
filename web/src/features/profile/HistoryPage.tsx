import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Star, Trophy, CircleX, Clock3, ChevronRight } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { profileApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { DifficultyBadge } from "@/shared/ui/badges";
import type { HistoryEntry } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

/** Pull stars/success out of the loosely-typed history result payload. */
function resultOf(entry: HistoryEntry): { success?: boolean; stars?: number } {
  const r = entry.result;
  if (r && typeof r === "object") {
    const o = r as Record<string, unknown>;
    return {
      success: typeof o.success === "boolean" ? o.success : undefined,
      stars: typeof o.stars === "number" ? o.stars : undefined,
    };
  }
  return {};
}

export function HistoryPage() {
  const { t } = useI18n();
  const history = useQuery({
    queryKey: ["profile", "history"],
    queryFn: profileApi.history,
  });

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("history.title")}</h1>
      </header>

      {history.isPending && (
        <div className="panel">
          <SkeletonRows rows={6} />
        </div>
      )}
      {history.isError && (
        <div className="panel">
          <ErrorState error={history.error} onRetry={() => history.refetch()} />
        </div>
      )}
      {history.isSuccess && history.data.length === 0 && (
        <div className="panel">
          <EmptyState title={t("history.empty")} />
        </div>
      )}

      {history.isSuccess && history.data.length > 0 && (() => {
        // Split into open (resumable) and closed (completed/failed) cases —
        // a case-file framing reads more like reconstructing an archive than
        // one flat, undifferentiated activity feed, and it surfaces "what
        // can I still act on" before "what's already settled" without
        // requiring any new data beyond what this query already returns.
        const open = history.data.filter(
          (e) => e.status !== "completed" && e.status !== "failed",
        );
        const closed = history.data.filter(
          (e) => e.status === "completed" || e.status === "failed",
        );
        return (
          <>
            {open.length > 0 && (
              <section aria-label={t("history.openCases")} style={{ marginBottom: 22 }}>
                <div className="band-title">{t("history.openCases")}</div>
                <div className="archive-grid">
                  {open.map((entry) => (
                    <HistoryCard key={entry.mission_id} entry={entry} t={t} />
                  ))}
                </div>
              </section>
            )}
            {closed.length > 0 && (
              <section aria-label={t("history.closedCases")}>
                <div className="band-title">{t("history.closedCases")}</div>
                <div className="archive-grid">
                  {closed.map((entry) => (
                    <HistoryCard key={entry.mission_id} entry={entry} t={t} />
                  ))}
                </div>
              </section>
            )}
          </>
        );
      })()}
    </div>
  );
}

function HistoryCard({
  entry,
  t,
}: {
  entry: HistoryEntry;
  t: (k: TranslationKey, p?: Record<string, string | number>) => string;
}) {
  const { success, stars } = resultOf(entry);
  const isDone = entry.status === "completed" || entry.status === "failed";
  const won = success === true || entry.status === "completed";
  // Completed/failed missions open the result page; others resume.
  const to = isDone
    ? `/app/missions/${entry.mission_id}/result`
    : `/app/missions/${entry.mission_id}`;
  return (
    <Link to={to} className="archive-card">
      <div className="ac-head">
        <span
          className={`ac-badge ${won ? "won" : entry.status === "failed" ? "lost" : "open"}`}
          aria-hidden
        >
          {entry.status === "failed" ? (
            <CircleX size={16} />
          ) : won ? (
            <Trophy size={16} />
          ) : (
            <Clock3 size={16} />
          )}
        </span>
        <div className="grow" style={{ minWidth: 0 }}>
          <div className="ac-title">{entry.title}</div>
          <div className="sub">
            {t(`type.${entry.type}` as TranslationKey)} ·{" "}
            <span className="mono-num">
              {new Date(entry.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
        <ChevronRight size={15} className="rtl-flip faint" aria-hidden />
      </div>
      <div className="ac-foot">
        <DifficultyBadge difficulty={entry.difficulty} />
        {typeof stars === "number" ? (
          <span className="ac-stars" aria-label={`${stars} / 5`}>
            {[1, 2, 3, 4, 5].map((n) => (
              <Star key={n} size={13} className={n <= stars ? "on" : "off"} aria-hidden />
            ))}
          </span>
        ) : (
          <span className="chip">
            {t(`missions.status.${entry.status}` as TranslationKey)}
          </span>
        )}
      </div>
    </Link>
  );
}
