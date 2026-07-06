import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Navigation, Map, Coins, Users, Search, Trophy } from "lucide-react";
import { useI18n, type TranslationKey } from "@/shared/i18n";
import { walletApi } from "@/shared/api/endpoints";
import type { RecommendedAction } from "@/shared/types/api";

/**
 * NextActionCard — the roadmap answer to "what should I do next and why?".
 * Shows the recommended action, the reason it matters, a priority chip, the
 * coin cost when the action is paid, and a deep link to the map/target.
 */

const actionIcons: Record<string, typeof Navigation> = {
  visit_location: Map,
  talk_to_character: Users,
  search_location: Search,
  review_clues: Search,
  prepare_final: Trophy,
};

/** Server pricing keys for paid action types; unlisted actions are free. */
const pricingKeyFor: Record<string, string> = {
  talk_to_character: "character_chat",
  search_location: "location_search",
  prepare_final: "final_judgment",
};

export function NextActionCard({
  action,
  missionId,
}: {
  action: RecommendedAction;
  missionId: string;
}) {
  const { t } = useI18n();
  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
    staleTime: 5 * 60_000,
  });

  const Icon = actionIcons[action.type] ?? Navigation;
  const priority = action.priority ?? "medium";
  const cost = pricing.data?.[pricingKeyFor[action.type] ?? ""] ?? 0;

  const target =
    action.target_type === "location" && action.target_id
      ? `/app/missions/${missionId}/locations/${action.target_id}`
      : action.target_type === "character" && action.target_id
        ? `/app/missions/${missionId}/characters/${action.target_id}`
        : `/app/missions/${missionId}/map`;

  return (
    <Link to={target} className={`next-action-card prio-${priority}`}>
      <span className="na-icon" aria-hidden>
        <Icon size={18} />
      </span>
      <div className="grow" style={{ minWidth: 0 }}>
        <div className="na-meta">
          <span className="faint" style={{ fontSize: 11 }}>
            {t("hud.recommended")}
          </span>
          <span className={`na-prio prio-${priority}`}>
            {t(`nextAction.priority.${priority}` as TranslationKey)}
          </span>
          {cost > 0 && (
            <span className="chip chip-wallet mono-num" title={t("wallet.title")}>
              <Coins size={11} aria-hidden />
              {cost}
            </span>
          )}
        </div>
        <strong className="na-title" style={{ unicodeBidi: "plaintext" }}>
          {action.title}
        </strong>
        {action.description && (
          <div className="na-reason" style={{ unicodeBidi: "plaintext" }}>
            {action.description}
          </div>
        )}
      </div>
      <span className="na-go" aria-hidden>
        <Map size={15} />
        {t("nextAction.openMap")}
      </span>
    </Link>
  );
}
