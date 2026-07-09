import { describe, expect, it, vi } from "vitest";
import { mockRequest } from "./mockClient";

describe("mock gameplay endpoints", () => {
  it("serves gameplay-status, preview, reports and board art", async () => {
        const created = await mockRequest<{ mission: { id: string } }>(
      "/api/v1/missions",
      { method: "POST", body: { type: "detective", difficulty: "medium" } },
    );
    const id = created.mission.id;

    // Skip past the mock's 8s "generating" window.
    const realNow = Date.now.bind(Date);
    vi.spyOn(Date, "now").mockImplementation(() => realNow() + 10_000);

    const status = await mockRequest<Record<string, unknown>>(
      `/api/v1/missions/${id}/gameplay-status`,
      { method: "GET" },
    );
    expect(Array.isArray(status.stages)).toBe(true);
    expect((status.stages as unknown[]).length).toBe(6);

    const preview = await mockRequest<{ time_cost_minutes: number }>(
      `/api/v1/missions/${id}/actions/preview`,
      { method: "POST", body: { action: "location_action" } },
    );
    expect(preview.time_cost_minutes).toBe(20);

    const report = await mockRequest<{ verdict: string }>(
      `/api/v1/missions/${id}/reports`,
      { method: "POST", body: { type: "progress_report", summary: "hi" } },
    );
    expect(report.verdict).toBe("accepted");

    const board = await mockRequest<{ board: { status: string } }>(
      `/api/v1/missions/${id}/art/board/generate`,
      { method: "POST", body: { board_type: "mission_board_background" } },
    );
    expect(board.board.status).toBe("ready");
    vi.restoreAllMocks();
  }, 15_000);
});
