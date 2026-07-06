import { Link } from "react-router-dom";
import {
  Search,
  MapPin,
  Lightbulb,
  Clock3,
  Radio,
  Trophy,
  CircleX,
  Footprints,
  CircleDot,
} from "lucide-react";
import { useI18n, type TranslationKey } from "@/shared/i18n";
import type { MissionEvent } from "@/shared/types/api";

/**
 * TimelineLog — a mission-log timeline so the player can see how far they have
 * progressed: mission time, event title, type, importance, and the related
 * location/clue when the event references one.
 */

type Importance = "high" | "medium" | "low";

const iconFor: Record<string, typeof Search> = {
  clue_discovered: Search,
  clue_inspected: Search,
  location_visited: MapPin,
  location_discovered: MapPin,
  fact_discovered: Lightbulb,
  time_advanced: Clock3,
  ai_guidance: Radio,
  director_hint: Radio,
  player_action: Footprints,
  mission_completed: Trophy,
  mission_ready: Trophy,
  mission_failed: CircleX,
};

const importanceFor: Record<string, Importance> = {
  clue_discovered: "high",
  mission_completed: "high",
  mission_failed: "high",
  mission_ready: "high",
  fact_discovered: "medium",
  time_advanced: "medium",
  location_visited: "medium",
  clue_inspected: "medium",
};

function str(payload: Record<string, unknown>, ...keys: string[]): string {
  for (const k of keys) {
    const v = payload[k];
    if (typeof v === "string" && v.trim()) return v;
  }
  return "";
}

export function TimelineLog({
  events,
  missionId,
  limit,
}: {
  events: MissionEvent[] | undefined;
  missionId: string;
  limit?: number;
}) {
  const { t } = useI18n();
  const list = (events ?? []).slice(0, limit ?? events?.length ?? 0);

  if (list.length === 0) {
    return <div className="tl-empty faint">{t("timeline.empty")}</div>;
  }

  const typeLabel = (type: string) => {
    const key = `event.${type}` as TranslationKey;
    const label = t(key);
    return label === key ? type.replace(/_/g, " ") : label;
  };

  return (
    <ol className="timeline-log">
      {list.map((ev) => {
        const payload = (ev.payload ?? {}) as Record<string, unknown>;
        const Icon = iconFor[ev.type] ?? CircleDot;
        const importance: Importance =
          (payload.importance as Importance) ?? importanceFor[ev.type] ?? "low";
        const title =
          str(payload, "title", "name", "fact", "summary", "hint") ||
          typeLabel(ev.type);
        const related = str(payload, "location_name", "clue_title", "source");
        const clock = str(payload, "new_time", "mission_time");
        const time =
          clock || new Date(ev.created_at).toLocaleTimeString(undefined, {
            hour: "2-digit",
            minute: "2-digit",
          });

        return (
          <li key={ev.id} className={`tl-item imp-${importance}`}>
            <span className="tl-rail" aria-hidden>
              <span className="tl-dot">
                <Icon size={12} />
              </span>
            </span>
            <div className="tl-body">
              <div className="tl-meta">
                <span className="tl-time mono-num">{time}</span>
                <span className="tl-type">{typeLabel(ev.type)}</span>
                <span className={`tl-imp imp-${importance}`}>
                  {t(`timeline.importance.${importance}` as TranslationKey)}
                </span>
              </div>
              <div className="tl-title" style={{ unicodeBidi: "plaintext" }}>
                {title}
              </div>
              {related && (
                <div className="tl-related">
                  <MapPin size={11} aria-hidden />
                  {related}
                </div>
              )}
            </div>
          </li>
        );
      })}
      {missionId && (
        <li className="tl-more">
          <Link className="faint" to={`/app/missions/${missionId}/events`}>
            {t("timeline.viewAll")} →
          </Link>
        </li>
      )}
    </ol>
  );
}
