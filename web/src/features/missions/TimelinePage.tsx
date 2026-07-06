import { useParams, Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Radio, ScrollText } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { useMissionStream } from "@/shared/api/sse";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { TimelineLog } from "./TimelineLog";
import { env } from "@/shared/config/env";

/**
 * TimelinePage — the full curated mission story, oldest first, live-updating.
 * The raw /events feed stays available for development via Diagnostics.
 */
export function TimelinePage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const timeline = useQuery({
    queryKey: ["mission", missionId, "timeline"],
    queryFn: () => missionsApi.timeline(missionId!),
    enabled: !!missionId,
  });

  const streamStatus = useMissionStream(missionId, {
    onEvent: () => {
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "timeline"],
      });
    },
  });

  const statusLabel =
    streamStatus === "live"
      ? t("events.live")
      : streamStatus === "polling"
        ? t("events.polling")
        : t("events.disconnected");

  return (
    <div className="page">
      <header className="page-header">
        <h1 className="row" style={{ gap: 10 }}>
          <ScrollText size={22} aria-hidden />
          {t("timeline.title")}
        </h1>
        <div className="row" style={{ gap: 8 }}>
          <span
            className={`chip ${streamStatus === "disconnected" ? "chip-danger" : "chip-mission"}`}
          >
            <Radio size={12} aria-hidden />
            {statusLabel}
          </span>
          {env.enableDiagnostics && missionId && (
            <Link className="chip" to={`/app/missions/${missionId}/events`}>
              {t("timeline.rawFeed")}
            </Link>
          )}
        </div>
      </header>

      <div className="panel" style={{ padding: 16 }}>
        {timeline.isPending && <SkeletonRows rows={7} />}
        {timeline.isError && (
          <ErrorState error={timeline.error} onRetry={() => timeline.refetch()} />
        )}
        {timeline.isSuccess && timeline.data.items.length === 0 && (
          <EmptyState title={t("timeline.empty")} />
        )}
        {timeline.isSuccess && timeline.data.items.length > 0 && missionId && (
          <TimelineLog
            items={timeline.data.items}
            missionId={missionId}
            newestFirst={false}
          />
        )}
      </div>
    </div>
  );
}
