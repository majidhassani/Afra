import { Link } from "react-router-dom";
import {
  Search,
  MapPin,
  Clock3,
  Trophy,
  CircleX,
  CircleDot,
  CircleCheck,
  Users,
  Sparkles,
  Flag,
  AlertTriangle,
  Unlock,
  Globe,
} from "lucide-react";
import { useI18n, type TranslationKey } from "@/shared/i18n";
import type { TimelineItem, TimelineItemType } from "@/shared/types/api";

/**
 * TimelineLog — the curated mission story (GET /missions/{id}/timeline).
 * Each item answers: what happened, when, where, and why it matters —
 * icon by category, importance color, and deep links to the related
 * location or clue.
 */

const iconFor: Record<TimelineItemType, typeof Search> = {
  mission_started: Flag,
  location_visited: MapPin,
  clue_discovered: Search,
  clue_inspected: Search,
  character_talked: Users,
  ai_guidance_received: Sparkles,
  time_advanced: Clock3,
  risk_changed: AlertTriangle,
  objective_completed: CircleCheck,
  objective_failed: CircleX,
  new_location_unlocked: Unlock,
  mission_ready_to_complete: Trophy,
  mission_completed: Trophy,
  mission_failed: CircleX,
  world_event: Globe,
};

export function TimelineLog({
  items,
  missionId,
  limit,
  newestFirst = true,
}: {
  items: TimelineItem[] | undefined;
  missionId: string;
  limit?: number;
  /** Preview panels show the latest items first; the full page reads as a story. */
  newestFirst?: boolean;
}) {
  const { t } = useI18n();
  let list = items ?? [];
  if (newestFirst) list = [...list].reverse();
  if (limit) list = list.slice(0, limit);

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
      {list.map((item, index) => {
        const Icon = iconFor[item.type] ?? CircleDot;
        const time =
          item.mission_time ||
          new Date(item.occurred_at).toLocaleTimeString(undefined, {
            hour: "2-digit",
            minute: "2-digit",
          });
        const linkTo = item.related_clue_id
          ? `/app/missions/${missionId}/clues/${item.related_clue_id}`
          : item.related_character_id
            ? `/app/missions/${missionId}/characters/${item.related_character_id}`
            : item.location_id
              ? `/app/missions/${missionId}/locations/${item.location_id}`
              : null;

        const body = (
          <div className="tl-body">
            <div className="tl-meta">
              <span className="tl-time mono-num">{time}</span>
              <span className="tl-type">{typeLabel(item.type)}</span>
              <span className={`tl-imp imp-${item.importance}`}>
                {t(`timeline.importance.${item.importance}` as TranslationKey)}
              </span>
            </div>
            <div className="tl-title" style={{ unicodeBidi: "plaintext" }}>
              {item.title}
            </div>
            {item.description && (
              <div className="tl-related" style={{ unicodeBidi: "plaintext" }}>
                {item.description}
              </div>
            )}
          </div>
        );

        return (
          <li
            key={item.id}
            className={`tl-item tl-reveal imp-${item.importance}`}
            style={{ animationDelay: `${Math.min(index, 8) * 45}ms` }}
          >
            <span className="tl-rail" aria-hidden>
              <span className="tl-dot">
                <Icon size={12} />
              </span>
            </span>
            {linkTo ? (
              <Link to={linkTo} className="tl-link grow">
                {body}
              </Link>
            ) : (
              body
            )}
          </li>
        );
      })}
      {missionId && limit && (
        <li className="tl-more">
          <Link className="faint" to={`/app/missions/${missionId}/timeline`}>
            {t("timeline.viewAll")} →
          </Link>
        </li>
      )}
    </ol>
  );
}
