import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { MessageSquare } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { charactersApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { AvatarPlaceholder, Meter } from "@/shared/ui/badges";

export function CharactersPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const characters = useQuery({
    queryKey: ["mission", missionId, "characters"],
    queryFn: () => charactersApi.list(missionId!),
    enabled: !!missionId,
  });

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("chars.title")}</h1>
      </header>
      <div className="panel">
        {characters.isPending && <SkeletonRows rows={5} />}
        {characters.isError && (
          <ErrorState
            error={characters.error}
            onRetry={() => characters.refetch()}
          />
        )}
        {characters.isSuccess && characters.data.length === 0 && (
          <EmptyState title={t("chars.empty")} />
        )}
        <div className="item-list">
          {characters.data?.map((c) => (
            <Link
              key={c.id}
              className="item-row"
              to={`/app/missions/${missionId}/characters/${c.id}`}
            >
              <AvatarPlaceholder name={c.name} prompt={c.avatar_prompt} />
              <div className="grow">
                <div className="title">{c.name}</div>
                <div className="sub">
                  {c.role} · {c.category}
                </div>
              </div>
              <div className="row" style={{ gap: 6 }}>
                <span className="faint">{t("chars.trust")}</span>
                <Meter value={c.trust_level} />
              </div>
              <span className="chip">{c.mood}</span>
              <MessageSquare size={15} aria-hidden />
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
