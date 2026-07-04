import { afterEach, describe, expect, it, vi } from "vitest";

// Pin the runtime config so these tests always exercise the real HTTP path,
// regardless of any local .env.local mock-mode override.
vi.mock("@/shared/config/env", () => ({
  env: {
    apiBaseUrl: "http://localhost:8080",
    enableMocks: false,
    defaultLanguage: "en",
    appVersion: "test",
  },
}));

import { ApiError, apiRequest, errorKey } from "./client";

function mockFetchOnce(status: number, body: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => ({
      ok: status >= 200 && status < 300,
      status,
      statusText: `HTTP ${status}`,
      json: async () => body,
    })),
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe("apiRequest envelope handling", () => {
  it("unwraps the data envelope on success", async () => {
    mockFetchOnce(200, { data: { wallet: { balance: 42 } } });
    const result = await apiRequest<{ wallet: { balance: number } }>(
      "/api/v1/wallet",
    );
    expect(result.wallet.balance).toBe(42);
  });

  it("throws a typed ApiError from the error envelope", async () => {
    mockFetchOnce(409, {
      error: { code: "insufficient_balance", message: "not enough coins" },
    });
    const err = await apiRequest("/api/v1/missions", { method: "POST", body: {} })
      .then(() => null)
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).code).toBe("insufficient_balance");
    expect((err as ApiError).status).toBe(409);
  });

  it("maps network failures to a network ApiError", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("failed to fetch");
      }),
    );
    const err = await apiRequest("/api/v1/wallet")
      .then(() => null)
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect((err as ApiError).isNetwork).toBe(true);
  });
});

describe("errorKey", () => {
  it("maps error kinds to translation keys", () => {
    expect(errorKey(new ApiError("network_error", "x", 0))).toBe("error.network");
    expect(errorKey(new ApiError("insufficient_balance", "x", 409))).toBe(
      "error.insufficient_balance",
    );
    expect(errorKey(new ApiError("mission_generating", "x", 409))).toBe(
      "error.mission_generating",
    );
    expect(errorKey(new ApiError("not_found", "x", 404))).toBe("error.not_found");
    expect(errorKey(new ApiError("conflict", "x", 409))).toBe("error.conflict");
    expect(errorKey(new ApiError("internal", "x", 500))).toBe("error.server");
    expect(errorKey(new Error("plain"))).toBe("error.server");
  });
});
