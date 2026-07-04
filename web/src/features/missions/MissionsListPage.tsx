import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { MissionRow } from "./MissionCard";

export function MissionsListPage() {
  const { t } = useI18n();
  const missions = useQuery({ queryKey: ["missions"], queryFn: missionsApi.list });

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("missions.title")}</h1>
        <Link className="btn btn-primary" to="/app/missions/new">
          <Plus size={15} aria-hidden />
          {t("dash.newMission")}
        </Link>
      </header>
      <div className="panel">
        {missions.isPending && <SkeletonRows rows={6} />}
        {missions.isError && (
          <ErrorState error={missions.error} onRetry={() => missions.refetch()} />
        )}
        {missions.isSuccess && missions.data.length === 0 && (
          <EmptyState
            title={t("missions.empty")}
            body={t("missions.empty.body")}
            action={
              <Link className="btn btn-secondary" to="/app/missions/new">
                <Plus size={14} aria-hidden />
                {t("dash.newMission")}
              </Link>
            }
          />
        )}
        <div className="item-list">
          {missions.data?.map((m) => (
            <MissionRow key={m.id} mission={m} />
          ))}
        </div>
      </div>
    </div>
  );
}
