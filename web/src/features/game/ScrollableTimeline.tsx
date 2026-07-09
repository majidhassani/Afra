import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronUp, ChevronDown, Radio } from "lucide-react";
import { missionsApi } from "@/shared/api/endpoints";
import { useI18n } from "@/shared/i18n";
import type { TimelineItem } from "@/shared/types/api";

/**
 * The everywhere-timeline: a collapsible bottom drawer with the curated
 * mission story feed and a current-time indicator. Mounted by the AppShell
 * on every mission screen so the story is always one tap away.
 */
export function ScrollableTimeline({
  missionId,
  currentTime,
}: {
  missionId: string;
  currentTime: string | undefined;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const timeline = useQuery({
    queryKey: ["timeline", missionId],
    queryFn: () => missionsApi.timeline(missionId),
    enabled: !!missionId && open,
    refetchInterval: open ? 20_000 : false,
  });

  return (
    <div className={`timeline-drawer${open ? " open" : ""}`}>
      <button
        className="timeline-drawer-handle"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
      >
        <Radio size={14} aria-hidden />
        <span>{t("game.timeline")}</span>
        {currentTime && (
          <span className="timeline-now">
            {t("game.timeline.now", { time: currentTime })}
          </span>
        )}
        {open ? <ChevronDown size={15} aria-hidden /> : <ChevronUp size={15} aria-hidden />}
      </button>
      {open && (
        <div className="timeline-drawer-body">
          {(timeline.data?.items ?? []).map((item) => (
            <TimelineRow key={item.id} item={item} />
          ))}
          {timeline.data && timeline.data.items.length === 0 && (
            <div className="muted" style={{ padding: 12 }}>
              {t("common.empty.title")}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function TimelineRow({ item }: { item: TimelineItem }) {
  return (
    <div className={`timeline-row imp-${item.importance}`}>
      <span className="timeline-dot" aria-hidden />
      <div>
        <div className="timeline-row-title">{item.title}</div>
        {item.description && (
          <div className="timeline-row-desc">{item.description}</div>
        )}
      </div>
      {item.mission_time && (
        <span className="timeline-row-time">{item.mission_time}</span>
      )}
    </div>
  );
}
