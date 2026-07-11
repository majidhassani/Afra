import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { cluesApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { Meter } from "@/shared/ui/badges";
import type { TranslationKey } from "@/shared/i18n/en";

export function importanceChip(importance: string): string {
  if (importance === "critical") return "chip-danger";
  if (importance === "high") return "chip-wallet";
  return "";
}

export function importanceLabel(
  t: (k: TranslationKey) => string,
  importance: string,
): string {
  const key = `clues.importance.${importance}` as TranslationKey;
  const label = t(key);
  return label === key ? importance : label;
}

export function CluesPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const clues = useQuery({
    queryKey: ["mission", missionId, "clues"],
    queryFn: () => cluesApi.list(missionId!),
    enabled: !!missionId,
  });

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("clues.title")}</h1>
      </header>
      <div className="panel">
        {clues.isPending && <SkeletonRows rows={5} />}
        {clues.isError && (
          <ErrorState error={clues.error} onRetry={() => clues.refetch()} />
        )}
        {clues.isSuccess && clues.data.length === 0 && (
          <EmptyState title={t("clues.empty")} body={t("clues.empty.body")} />
        )}
        <div className="item-list">
          {clues.data?.map((clue) => (
            <Link
              key={clue.id}
              className="item-row"
              to={`/app/missions/${missionId}/clues/${clue.id}`}
            >
              <Search size={15} aria-hidden />
              <div className="grow">
                <div className="title">{clue.title}</div>
                <div className="sub">{clue.short_description}</div>
              </div>
              <span className={`chip ${importanceChip(clue.importance)}`}>
                {importanceLabel(t, clue.importance)}
              </span>
              <span className="row faint" style={{ gap: 6 }}>
                {t("clues.reliability")}
                {/* Cyan, not gold — reliability is an AI-assessed
                    confidence score (same family as character trust_level
                    in CharactersPage.tsx), not a monetary value. Gold is
                    reserved for wallet/reward amounts. */}
                <Meter value={clue.reliability} color="var(--accent-ai)" />
              </span>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
