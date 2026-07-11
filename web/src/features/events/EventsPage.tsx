import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Radio,
  MapPin,
  Search,
  Users,
  Clock3,
  Sparkles,
  CircleDot,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi } from "@/shared/api/endpoints";
import { useMissionStream } from "@/shared/api/sse";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";

const eventIcons: Array<[RegExp, typeof Radio]> = [
  [/location/, MapPin],
  [/clue/, Search],
  [/character|chat/, Users],
  [/time/, Clock3],
  [/guidance|ai/, Sparkles],
];

function iconFor(type: string) {
  for (const [pattern, Icon] of eventIcons) {
    if (pattern.test(type)) return Icon;
  }
  return CircleDot;
}

function payloadSummary(payload: Record<string, unknown>): string {
  const parts = Object.entries(payload)
    .filter(([, v]) => typeof v === "string" || typeof v === "number")
    .slice(0, 3)
    .map(([k, v]) => `${k}: ${String(v)}`);
  return parts.join(" · ");
}

export function EventsPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const events = useQuery({
    queryKey: ["mission", missionId, "events"],
    queryFn: () => missionsApi.events(missionId!),
    enabled: !!missionId,
  });

  const streamStatus = useMissionStream(missionId, {
    onEvent: () => {
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "events"],
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
        <h1>{t("events.title")}</h1>
        <span
          className={`chip ${
            streamStatus === "disconnected"
              ? "chip-danger"
              : streamStatus === "polling"
                ? "chip-warning"
                : "chip-mission"
          }`}
        >
          <Radio size={12} aria-hidden />
          {statusLabel}
        </span>
      </header>

      <div className="panel">
        {events.isPending && <SkeletonRows rows={7} />}
        {events.isError && (
          <ErrorState error={events.error} onRetry={() => events.refetch()} />
        )}
        {events.isSuccess && events.data.length === 0 && (
          <EmptyState title={t("events.empty")} />
        )}
        <div className="timeline">
          {events.data?.map((ev) => {
            const Icon = iconFor(ev.type);
            return (
              <div key={ev.id} className="timeline-item">
                <span className="timeline-icon">
                  <Icon size={13} aria-hidden />
                </span>
                <div className="grow" style={{ minWidth: 0 }}>
                  <div style={{ fontWeight: 500 }}>
                    {ev.type.replace(/_/g, " ")}
                  </div>
                  {ev.payload && (
                    <div className="sub" style={{ unicodeBidi: "plaintext" }}>
                      {payloadSummary(ev.payload)}
                    </div>
                  )}
                </div>
                <span className="faint mono-num">
                  {new Date(ev.created_at).toLocaleTimeString()}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
