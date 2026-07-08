import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { MessageSquare, MapPin } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { charactersApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { Avatar } from "@/shared/ui/Avatar";
import { Meter } from "@/shared/ui/badges";

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
        <div>
          <h1>{t("chars.title")}</h1>
          <p className="subtitle">{t("chars.subtitle")}</p>
        </div>
      </header>

      {characters.isPending && (
        <div className="panel">
          <SkeletonRows rows={5} />
        </div>
      )}
      {characters.isError && (
        <div className="panel">
          <ErrorState
            error={characters.error}
            onRetry={() => characters.refetch()}
          />
        </div>
      )}
      {characters.isSuccess && characters.data.length === 0 && (
        <div className="panel">
          <EmptyState title={t("chars.empty")} body={t("chars.empty.body")} />
        </div>
      )}

      <div className="card-grid">
        {characters.data?.map((c) => (
          <Link
            key={c.id}
            className="char-card"
            to={`/app/missions/${missionId}/characters/${c.id}`}
          >
            <div className="char-card-head">
              <Avatar
                name={c.name}
                category={c.category}
                imageUrl={c.avatar_url || undefined}
                version={c.avatar_version}
                size="lg"
              />
              <div className="grow" style={{ minWidth: 0 }}>
                <div className="char-card-name">{c.name}</div>
                <div className="sub">{c.role}</div>
              </div>
              <span className={`status-chip cat-${c.category}`}>{c.category}</span>
            </div>
            <div className="char-card-meters">
              <div className="char-meter">
                <span className="faint">{t("chars.trust")}</span>
                <Meter value={c.trust_level} color="var(--accent-ai)" />
              </div>
              <div className="row" style={{ gap: 6, flexWrap: "wrap" }}>
                <span className="status-chip">{t("chars.mood")}: {c.mood}</span>
                {c.current_location_id && (
                  <span className="status-chip">
                    <MapPin size={11} aria-hidden />
                    {t("chars.onSite")}
                  </span>
                )}
              </div>
            </div>
            <div className="char-card-cta">
              <span className="game-btn game-btn-ghost sm">
                <MessageSquare size={14} aria-hidden />
                {t("chars.talk")}
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
