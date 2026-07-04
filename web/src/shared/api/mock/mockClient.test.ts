import { describe, expect, it } from "vitest";
import { mockRequest } from "./mockClient";
import type {
  MissionDashboard,
  Mission,
  Wallet,
} from "@/shared/types/api";

describe("mock adapter", () => {
  it("serves the seeded mission list", async () => {
    const res = await mockRequest<{ missions: Mission[] }>("/api/v1/missions", {});
    expect(res.missions.length).toBeGreaterThan(0);
    expect(res.missions[0].title).toBeTruthy();
  });

  it("serves a mission dashboard with public fields only", async () => {
    const { missions } = await mockRequest<{ missions: Mission[] }>(
      "/api/v1/missions",
      {},
    );
    const dashboard = await mockRequest<MissionDashboard>(
      `/api/v1/missions/${missions[0].id}`,
      {},
    );
    expect(dashboard.characters.length).toBeGreaterThan(0);
    expect(dashboard.clues.length).toBeGreaterThan(0);
    const serialized = JSON.stringify(dashboard);
    // Privacy rule: truth-layer fields must never appear in player payloads.
    expect(serialized).not.toContain("internal_truth");
    expect(serialized).not.toContain("private_state");
    expect(serialized).not.toContain("hidden_state");
  });

  it("charges coins for paid actions", async () => {
    const before = await mockRequest<{ wallet: Wallet }>("/api/v1/wallet", {});
    const { missions } = await mockRequest<{ missions: Mission[] }>(
      "/api/v1/missions",
      {},
    );
    await mockRequest(`/api/v1/missions/${missions[0].id}/guidance`, {
      method: "POST",
      body: { message: "help" },
    });
    const after = await mockRequest<{ wallet: Wallet }>("/api/v1/wallet", {});
    expect(after.wallet.balance).toBeLessThan(before.wallet.balance);
  });

  it("404s unknown routes", async () => {
    await expect(mockRequest("/api/v1/nope", {})).rejects.toMatchObject({
      status: 404,
    });
  });
});
