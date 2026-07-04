import { apiRequest } from "./client";
import { env } from "@/shared/config/env";
import type {
  ActionResult,
  AuthResponse,
  ChatResult,
  CharacterDetail,
  ClueExplainResult,
  ClueInspectResult,
  CreateMissionRequest,
  GuidanceContext,
  GuidanceResult,
  HistoryEntry,
  JournalNote,
  LocationDetail,
  MapView,
  Mission,
  MissionDashboard,
  MissionEvent,
  Pricing,
  Profile,
  ProfileStats,
  PublicCharacter,
  PublicClue,
  TimeAdvanceResult,
  TimeInfo,
  TimeUnit,
  Transaction,
  User,
  Wallet,
} from "@/shared/types/api";

const V1 = "/api/v1";

export const authApi = {
  register: (input: { email: string; password: string; display_name: string }) =>
    apiRequest<AuthResponse>(`${V1}/auth/register`, {
      method: "POST",
      body: input,
      anonymous: true,
    }),
  login: (input: { email: string; password: string }) =>
    apiRequest<AuthResponse>(`${V1}/auth/login`, {
      method: "POST",
      body: input,
      anonymous: true,
    }),
  logout: (refresh_token: string) =>
    apiRequest<{ status: string }>(`${V1}/auth/logout`, {
      method: "POST",
      body: { refresh_token },
      anonymous: true,
    }),
  me: () => apiRequest<{ user: User }>(`${V1}/me`).then((r) => r.user),
};

export const profileApi = {
  get: () =>
    apiRequest<{ profile: Profile }>(`${V1}/profile`).then((r) => r.profile),
  update: (display_name: string) =>
    apiRequest<{ profile: Profile }>(`${V1}/profile`, {
      method: "PUT",
      body: { display_name },
    }).then((r) => r.profile),
  stats: () => apiRequest<ProfileStats>(`${V1}/profile/stats`),
  history: () =>
    apiRequest<{ history: HistoryEntry[] }>(`${V1}/profile/history`).then(
      (r) => r.history,
    ),
  badges: () =>
    apiRequest<{ badges: unknown }>(`${V1}/profile/badges`).then(
      (r) => r.badges,
    ),
};

export const walletApi = {
  get: () =>
    apiRequest<{ wallet: Wallet }>(`${V1}/wallet`).then((r) => r.wallet),
  transactions: (limit = 50) =>
    apiRequest<{ transactions: Transaction[] }>(
      `${V1}/wallet/transactions?limit=${limit}`,
    ).then((r) => r.transactions),
  pricing: () =>
    apiRequest<{ pricing: Pricing }>(`${V1}/wallet/pricing`).then(
      (r) => r.pricing,
    ),
  claimRewardedAd: () =>
    apiRequest<{ transaction: Transaction }>(`${V1}/wallet/rewarded-ad/claim`, {
      method: "POST",
    }),
  verifyPurchase: (input: {
    platform: "ios" | "android";
    product_id: string;
    receipt: string;
  }) =>
    apiRequest<{ transaction: Transaction }>(`${V1}/wallet/purchase/verify`, {
      method: "POST",
      body: input,
    }),
};

export const missionsApi = {
  list: () =>
    apiRequest<{ missions: Mission[] }>(`${V1}/missions`).then(
      (r) => r.missions,
    ),
  create: (input: CreateMissionRequest) =>
    apiRequest<{ mission: Mission }>(`${V1}/missions`, {
      method: "POST",
      body: input,
    }).then((r) => r.mission),
  dashboard: (missionId: string) =>
    apiRequest<MissionDashboard>(`${V1}/missions/${missionId}`),
  archive: (missionId: string) =>
    apiRequest<{ mission: Mission }>(`${V1}/missions/${missionId}/archive`, {
      method: "POST",
    }),
  events: (missionId: string, limit = 100) =>
    apiRequest<{ events: MissionEvent[] }>(
      `${V1}/missions/${missionId}/events?limit=${limit}`,
    ).then((r) => r.events),
};

export const mapApi = {
  view: (missionId: string) =>
    apiRequest<MapView>(`${V1}/missions/${missionId}/map`),
  location: (missionId: string, locationId: string) =>
    apiRequest<LocationDetail>(
      `${V1}/missions/${missionId}/locations/${locationId}`,
    ),
  runAction: (missionId: string, locationId: string, action: string) =>
    apiRequest<ActionResult>(
      `${V1}/missions/${missionId}/locations/${locationId}/actions`,
      { method: "POST", body: { action } },
    ),
  askAi: (missionId: string, locationId: string, message: string) =>
    apiRequest<GuidanceResult>(
      `${V1}/missions/${missionId}/locations/${locationId}/ask-ai`,
      { method: "POST", body: { message } },
    ),
};

export const charactersApi = {
  list: (missionId: string) =>
    apiRequest<{ characters: PublicCharacter[] }>(
      `${V1}/missions/${missionId}/characters`,
    ).then((r) => r.characters),
  detail: (missionId: string, characterId: string) =>
    apiRequest<CharacterDetail>(
      `${V1}/missions/${missionId}/characters/${characterId}`,
    ),
  chat: (
    missionId: string,
    characterId: string,
    message: string,
    locationId?: string,
  ) =>
    apiRequest<ChatResult>(
      `${V1}/missions/${missionId}/characters/${characterId}/chat`,
      {
        method: "POST",
        body: { message, ...(locationId ? { location_id: locationId } : {}) },
      },
    ),
};

export const cluesApi = {
  list: (missionId: string) =>
    apiRequest<{ clues: PublicClue[] }>(`${V1}/missions/${missionId}/clues`).then(
      (r) => r.clues,
    ),
  detail: (missionId: string, clueId: string) =>
    apiRequest<{ clue: PublicClue }>(
      `${V1}/missions/${missionId}/clues/${clueId}`,
    ).then((r) => r.clue),
  inspect: (missionId: string, clueId: string, question?: string) =>
    apiRequest<ClueInspectResult>(
      `${V1}/missions/${missionId}/clues/${clueId}/inspect`,
      { method: "POST", body: question ? { question } : {} },
    ),
  explain: (missionId: string, clueId: string) =>
    apiRequest<ClueExplainResult>(
      `${V1}/missions/${missionId}/clues/${clueId}/explain`,
      { method: "POST", body: {} },
    ),
};

export const guidanceApi = {
  ask: (missionId: string, message: string, context?: GuidanceContext) =>
    apiRequest<GuidanceResult>(`${V1}/missions/${missionId}/guidance`, {
      method: "POST",
      body: { message, ...(context ? { context } : {}) },
    }),
};

export const timeApi = {
  get: (missionId: string) =>
    apiRequest<TimeInfo>(`${V1}/missions/${missionId}/time`),
  advance: (missionId: string, amount: number, unit: TimeUnit) =>
    apiRequest<TimeAdvanceResult>(`${V1}/missions/${missionId}/time/advance`, {
      method: "POST",
      body: { amount, unit },
    }),
};

export const journalApi = {
  list: (missionId: string) =>
    apiRequest<{ notes: JournalNote[] }>(
      `${V1}/missions/${missionId}/journal`,
    ).then((r) => r.notes),
  create: (missionId: string, input: { title?: string; content: string }) =>
    apiRequest<{ note: JournalNote }>(`${V1}/missions/${missionId}/journal`, {
      method: "POST",
      body: input,
    }).then((r) => r.note),
  update: (
    missionId: string,
    noteId: string,
    input: { title?: string; content: string },
  ) =>
    apiRequest<{ note: JournalNote }>(
      `${V1}/missions/${missionId}/journal/${noteId}`,
      { method: "PUT", body: input },
    ).then((r) => r.note),
  remove: (missionId: string, noteId: string) =>
    apiRequest<void>(`${V1}/missions/${missionId}/journal/${noteId}`, {
      method: "DELETE",
    }),
};

export const healthApi = {
  /** /health and /ready live outside /api/v1 and return non-envelope bodies. */
  async check(path: "/health" | "/ready"): Promise<{
    ok: boolean;
    latencyMs: number;
  }> {
    const start = performance.now();
    try {
      if (env.enableMocks) {
        return { ok: true, latencyMs: 1 };
      }
      const res = await fetch(`${env.apiBaseUrl}${path}`);
      return { ok: res.ok, latencyMs: Math.round(performance.now() - start) };
    } catch {
      return { ok: false, latencyMs: Math.round(performance.now() - start) };
    }
  },
};
