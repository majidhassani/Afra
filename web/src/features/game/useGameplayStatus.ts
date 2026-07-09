import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { gameplayApi } from "@/shared/api/endpoints";
import { announceStageUpdate } from "./gameEvents";

/**
 * The one HUD query: GET /missions/{id}/gameplay-status. The backend runs a
 * lazy stage-engine pass on every read, so stage transitions earned by any
 * action surface here even if the action's own response was not inspected —
 * announce them as popups exactly once per payload.
 */
export function useGameplayStatus(missionId: string | undefined) {
  const query = useQuery({
    queryKey: ["gameplay-status", missionId],
    queryFn: () => gameplayApi.status(missionId!),
    enabled: !!missionId,
    refetchInterval: 30_000,
  });

  const stageUpdate = query.data?.stage_update;
  useEffect(() => {
    if (stageUpdate) announceStageUpdate(stageUpdate);
  }, [stageUpdate]);

  return query;
}

/** Invalidate everything a gameplay action can change. */
export function useInvalidateGameplay(missionId: string | undefined) {
  const qc = useQueryClient();
  return () => {
    if (!missionId) return;
    void qc.invalidateQueries({ queryKey: ["gameplay-status", missionId] });
    void qc.invalidateQueries({ queryKey: ["mission", missionId] });
    void qc.invalidateQueries({ queryKey: ["timeline", missionId] });
  };
}
