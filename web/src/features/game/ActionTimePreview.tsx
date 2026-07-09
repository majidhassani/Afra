import { useEffect, useState } from "react";
import { Clock3, TriangleAlert } from "lucide-react";
import { gameplayApi } from "@/shared/api/endpoints";
import { useI18n } from "@/shared/i18n";
import { GameButton } from "@/shared/ui/game";
import type { ActionPreview } from "@/shared/types/api";

/**
 * Bottom-sheet confirmation shown before a clocked action runs: the time
 * cost, coin cost, risk note, and what the mission clock will read after.
 * Time is a real resource — the player always sees the price first.
 */
export function ActionTimePreview({
  missionId,
  action,
  targetId,
  onConfirm,
  onCancel,
}: {
  missionId: string;
  /** Backend preview action type: travel | location_action | character_chat | clue_inspect | report_submit */
  action: string;
  targetId?: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const { t } = useI18n();
  const [preview, setPreview] = useState<ActionPreview | null>(null);

  useEffect(() => {
    let cancelled = false;
    gameplayApi
      .previewAction(missionId, action, targetId)
      .then((p) => {
        if (!cancelled) setPreview(p);
      })
      .catch(() => {
        // Preview is best-effort; never block the action on it.
        if (!cancelled) onConfirm();
      });
    return () => {
      cancelled = true;
    };
  }, [missionId, action, targetId]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!preview) return null;
  return (
    <div className="time-preview-backdrop" role="dialog" aria-modal="true">
      <div className="time-preview-sheet">
        <div className="time-preview-row">
          <Clock3 size={16} aria-hidden />
          <span>
            {t("game.timeCostLine", {
              minutes: preview.time_cost_minutes,
              time: preview.new_time_if_done,
            })}
          </span>
        </div>
        {preview.coin_cost > 0 && (
          <div className="time-preview-row muted">
            {t("game.coinCostLine", { coins: preview.coin_cost })}
          </div>
        )}
        {preview.risk_note && (
          <div className="time-preview-row risk">
            <TriangleAlert size={15} aria-hidden />
            <span>{preview.risk_note}</span>
          </div>
        )}
        <div className="time-preview-actions">
          <GameButton variant="ghost" onClick={onCancel}>
            {t("common.cancel")}
          </GameButton>
          <GameButton onClick={onConfirm}>{t("game.proceed")}</GameButton>
        </div>
      </div>
    </div>
  );
}
