import { describe, expect, it } from "vitest";
import { mockRequest } from "./mockClient";
import type {
  Mission,
  MapView,
  PublicClue,
  TimelineView,
  MissionResult,
} from "@/shared/types/api";

/**
 * E2E smoke over the mock adapter: the core mission loop the release checklist
 * calls out — list → dashboard → map → clue → chat → guidance → complete →
 * result → timeline — exercising the endpoints added in this transformation.
 */
describe("mission loop e2e (mock)", () => {
  it("walks create → map → clue → chat → guidance → complete → result → timeline", async () => {
    const { missions } = await mockRequest<{ missions: Mission[] }>(
      "/api/v1/missions",
      {},
    );
    const id = missions[0].id;

    // Map renders Google-Maps-compatible markers.
    const map = await mockRequest<MapView>(`/api/v1/missions/${id}/map`, {});
    expect(map.locations.length).toBeGreaterThan(0);

    // Clues carry the image contract (url + status), never a prompt.
    const { clues } = await mockRequest<{ clues: PublicClue[] }>(
      `/api/v1/missions/${id}/clues`,
      {},
    );
    expect(clues.length).toBeGreaterThan(0);
    expect(clues[0]).toHaveProperty("image_status");
    expect(JSON.stringify(clues)).not.toContain("prompt");

    // Generate a clue image → status becomes ready with a URL.
    const gen = await mockRequest<{ clue: PublicClue }>(
      `/api/v1/missions/${id}/clues/${clues[0].id}/image`,
      { method: "POST" },
    );
    expect(gen.clue.image_status).toBe("ready");
    expect(gen.clue.image_url).toBeTruthy();

    // Ask Mission Control (paid AI action).
    const guidance = await mockRequest<{ message: string }>(
      `/api/v1/missions/${id}/guidance`,
      { method: "POST", body: { message: "where next?" } },
    );
    expect(guidance.message).toBeTruthy();

    // Complete the mission → judged result.
    const result = await mockRequest<MissionResult>(
      `/api/v1/missions/${id}/complete`,
      { method: "POST", body: {} },
    );
    expect(result.result_title).toBeTruthy();
    expect(result.stars).toBeGreaterThanOrEqual(0);

    // Standalone result endpoint returns the persisted report.
    const stored = await mockRequest<{
      mission_status: string;
      result: MissionResult;
    }>(`/api/v1/missions/${id}/result`, {});
    expect(stored.mission_status).toBe("completed");
    expect(stored.result.score).toBe(result.score);

    // Curated timeline reflects the completed mission, no raw prompts.
    const timeline = await mockRequest<TimelineView>(
      `/api/v1/missions/${id}/timeline`,
      {},
    );
    expect(timeline.items.length).toBeGreaterThan(0);
    expect(timeline.items.some((i) => i.type === "mission_completed")).toBe(true);
    expect(JSON.stringify(timeline)).not.toContain("prompt");
  });
});
