import { useQuery, useQueryClient } from "@tanstack/react-query";
import { missionsApi } from "@/shared/api/endpoints";
import { useMissionStream } from "@/shared/api/sse";

/**
 * Mission dashboard query. Polls while the mission is generating and keeps a
 * live event stream open to invalidate mission data when the world changes.
 */
export function useMissionDashboard(missionId: string | undefined) {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["mission", missionId],
    queryFn: () => missionsApi.dashboard(missionId!),
    enabled: !!missionId,
    refetchInterval: (q) =>
      q.state.data?.mission.status === "generating" ? 2500 : false,
  });

  const streamStatus = useMissionStream(missionId, {
    enabled: !!missionId && query.data?.mission.status === "generating",
    onEvent: () => {
      void queryClient.invalidateQueries({ queryKey: ["mission", missionId] });
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "events"],
      });
    },
  });

  return { ...query, streamStatus };
}
