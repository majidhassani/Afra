import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { profileApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { MissionStatusBadge, DifficultyBadge } from "@/shared/ui/badges";
import type { TranslationKey } from "@/shared/i18n/en";

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
      <div className="panel">
        {history.isPending && <SkeletonRows rows={6} />}
        {history.isError && (
          <ErrorState error={history.error} onRetry={() => history.refetch()} />
        )}
        {history.isSuccess && history.data.length === 0 && (
          <EmptyState title={t("history.empty")} />
        )}
        <div className="item-list">
          {history.data?.map((entry) => (
            <Link
              key={entry.mission_id}
              className="item-row"
              to={`/app/missions/${entry.mission_id}`}
            >
              <div className="grow">
                <div className="title">{entry.title}</div>
                <div className="sub">
                  {t(`type.${entry.type}` as TranslationKey)} ·{" "}
                  <span className="mono-num">
                    {new Date(entry.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
              <DifficultyBadge difficulty={entry.difficulty} />
              <MissionStatusBadge status={entry.status} />
              <ChevronRight size={15} className="rtl-flip" aria-hidden />
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
