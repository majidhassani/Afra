import { Link } from "react-router-dom";
import { ChevronRight, MapPin } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import type { Mission } from "@/shared/types/api";
import { MissionStatusBadge, DifficultyBadge } from "@/shared/ui/badges";
import type { TranslationKey } from "@/shared/i18n/en";

export function MissionRow({ mission }: { mission: Mission }) {
  const { t } = useI18n();
  return (
    <Link className="item-row" to={`/app/missions/${mission.id}`}>
      <div className="grow">
        <div className="title">{mission.title || t(`type.${mission.type}` as TranslationKey)}</div>
        <div className="sub row" style={{ gap: 6 }}>
          <span>{t(`type.${mission.type}` as TranslationKey)}</span>
          {mission.region && (
            <>
              <MapPin size={11} aria-hidden />
              <span>{mission.region}</span>
            </>
          )}
        </div>
      </div>
      <DifficultyBadge difficulty={mission.difficulty} />
      <MissionStatusBadge status={mission.status} />
      <ChevronRight size={15} className="rtl-flip" aria-hidden />
    </Link>
  );
}
