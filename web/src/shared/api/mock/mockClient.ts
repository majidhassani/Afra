/**
 * Mock adapter for offline frontend development (VITE_ENABLE_MOCKS=true).
 *
 * Mirrors the backend's {"data": ...} envelope semantics at the apiRequest
 * level: mockRequest resolves with what would be inside `data`, or throws
 * ApiError like the real client. State lives in memory for the session.
 */
import type { RequestOptions } from "../client";
import type {
  Mission,
  MissionReport,
  ReportType,
  Stage,
  Transaction,
  User,
  Wallet,
} from "@/shared/types/api";
import {
  buildMissionBundle,
  MOCK_PRICING,
  mockId,
  type MockMissionBundle,
} from "./mockData";

/** Tiny SVG data-URL portrait used by the mock avatar/clue image endpoints. */
const MOCK_PORTRAIT =
  "data:image/svg+xml;utf8," +
  encodeURIComponent(
    `<svg xmlns='http://www.w3.org/2000/svg' width='128' height='128'><defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'><stop offset='0' stop-color='#1a1f27'/><stop offset='1' stop-color='#2c3036'/></linearGradient></defs><rect width='128' height='128' fill='url(#g)'/><circle cx='64' cy='50' r='24' fill='#7dd3c7' opacity='0.7'/><rect x='28' y='82' width='72' height='40' rx='18' fill='#7dd3c7' opacity='0.5'/></svg>`,
  );

/** Mirrors the backend DeriveWorldState so mock mode themes the shell too. */
function deriveMockWorldState(bundle: MockMissionBundle) {
  const ps = (bundle.mission.public_state ?? {}) as Record<string, unknown>;
  const weatherText = String(ps.weather ?? ps["جو"] ?? "").toLowerCase();
  const weather = /storm|طوفان/.test(weatherText)
    ? "storm"
    : /snow|برف/.test(weatherText)
      ? "snow"
      : /rain|بارانی|باران/.test(weatherText)
        ? "rain"
        : /fog|مه/.test(weatherText)
          ? "fog"
          : "clear";
  const hourMatch = /(\d{1,2}):(\d{2})/.exec(bundle.mission.current_time);
  const hour = hourMatch ? Number(hourMatch[1]) : 9;
  const tod = hour >= 20 || hour < 5 ? "night" : hour >= 17 ? "dusk" : "day";
  const biomeByType: Record<string, string> = {
    wildlife_rescue: "forest",
    exploration: "space",
    survival: "snow",
    disaster_response: "desert",
    diplomacy: "city",
    detective: "city",
    medical_mystery: "horror",
  };
  const biome = biomeByType[bundle.mission.type] ?? "city";
  const modifier =
    weather === "rain" || weather === "storm"
      ? "rain"
      : weather === "snow"
        ? "snow"
        : tod === "night"
          ? "night"
          : "day";
  return {
    mission_time: bundle.mission.current_time,
    weather,
    time_of_day: tod,
    visibility: weather === "clear" && tod === "day" ? "high" : "medium",
    risk_score: 20,
    urgency: "calm",
    world_phase: "investigation",
    danger: false,
    theme_id: `${biome}_${modifier}`,
    active_events: weather !== "clear" ? [`weather_${weather}`] : [],
  };
}

class MockApiError extends Error {
  code: string;
  status: number;
  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
  get isNetwork() {
    return false;
  }
  get isUnauthorized() {
    return this.status === 401;
  }
}

const now = () => new Date().toISOString();

interface MockState {
  user: User;
  wallet: Wallet;
  transactions: Transaction[];
  missions: Map<string, MockMissionBundle>;
  generatingUntil: Map<string, number>;
  adClaimedToday: boolean;
}

const state: MockState = {
  user: {
    id: "mock-user-1",
    email: "agent@agentverse.dev",
    display_name: "Agent Zero",
    created_at: now(),
  },
  wallet: {
    id: "mock-wallet-1",
    user_id: "mock-user-1",
    balance: 500,
    reserved_balance: 0,
    created_at: now(),
    updated_at: now(),
  },
  transactions: [],
  missions: new Map(),
  generatingUntil: new Map(),
  adClaimedToday: false,
};

/** Per-mission report history for the mock report center. */
const mockReports = new Map<string, MissionReport[]>();

/** Fixed action time costs mirroring internal/mission/timecost.go. */
const MOCK_TIME_COSTS: Record<string, number> = {
  travel: 15,
  location_action: 20,
  character_chat: 10,
  clue_inspect: 10,
  report_submit: 15,
};

/**
 * Deterministic mock stage list mirroring mission.DefaultStages, with
 * statuses derived from the bundle's current counts (monotonic, like the
 * backend engine).
 */
function mockStages(bundle: MockMissionBundle): Stage[] {
  const found = bundle.clues.length;
  const confirmed = bundle.clues.filter((c) => c.status === "confirmed").length;
  const visited = bundle.markers.filter((m) => m.status === "visited").length;
  const talked = bundle.events.filter((e) => e.type === "dialogue").length;
  const accepted = (t: string) =>
    (mockReports.get(bundle.mission.id) ?? []).filter(
      (r) => r.type === t && r.verdict === "accepted",
    ).length;
  const finished =
    bundle.mission.status === "completed" || bundle.mission.status === "failed";

  const defs: Array<Omit<Stage, "status" | "progress" | "found_clue_count">> = [
    {
      id: "arrival",
      title: "Arrival",
      description: "Get on site: open the map and visit your first location.",
      required_clue_count: 0,
      required_actions: [{ type: "visit_locations", count: 1, done: Math.min(visited, 1) }],
      reward: { xp: 25, coins: 10 },
      unlock_on_complete: { next_stage_id: "first_clues" },
    },
    {
      id: "first_clues",
      title: "First Clues",
      description: "Search the area and secure your first pieces of evidence.",
      required_clue_count: 2,
      required_actions: [{ type: "find_clues", count: 2, done: Math.min(found, 2) }],
      reward: { xp: 50, coins: 15 },
      unlock_on_complete: { next_stage_id: "identify_suspect", unlock_location: true },
    },
    {
      id: "identify_suspect",
      title: "Identify the Suspect",
      description: "Gather enough clues and testimony to name a prime suspect.",
      required_clue_count: 5,
      required_actions: [
        { type: "find_clues", count: 5, done: Math.min(found, 5) },
        { type: "interview_characters", count: 2, done: Math.min(talked, 2) },
        {
          type: "submit_report",
          report: "suspect_report",
          count: 1,
          done: Math.min(accepted("suspect_report"), 1),
        },
      ],
      reward: { xp: 100, coins: 25, badge: "sharp_eye" },
      unlock_on_complete: {
        next_stage_id: "confirm_evidence",
        unlock_location: true,
        reveal_suspect: true,
      },
    },
    {
      id: "confirm_evidence",
      title: "Confirm the Evidence",
      description: "Inspect and confirm the evidence that carries your case.",
      required_clue_count: 0,
      required_actions: [
        { type: "confirm_evidence", count: 3, done: Math.min(confirmed, 3) },
      ],
      reward: { xp: 100, coins: 25 },
      unlock_on_complete: { next_stage_id: "final_report" },
    },
    {
      id: "final_report",
      title: "Final Report",
      description: "You have what you need. Submit your final decision to command.",
      required_clue_count: 0,
      required_actions: [
        { type: "final_decision", count: 1, done: finished ? 1 : 0 },
      ],
      reward: { xp: 150, coins: 50 },
      unlock_on_complete: { next_stage_id: "debrief" },
    },
    {
      id: "debrief",
      title: "Debrief",
      description: "Review the outcome, rewards, and what you missed.",
      required_clue_count: 0,
      required_actions: [],
      reward: { xp: 0, coins: 0 },
      unlock_on_complete: {},
    },
  ];

  let gate = true; // stages before the gate-breaking one are completable
  return defs.map((def) => {
    const need = def.required_actions.reduce((s, a) => s + a.count, 0);
    const done = def.required_actions.reduce((s, a) => s + a.done, 0);
    const complete = need === 0 ? finished : done >= need;
    let status: Stage["status"];
    if (gate && complete) {
      status = "completed";
    } else if (gate) {
      status = "active";
      gate = false;
    } else {
      status = "locked";
    }
    return {
      ...def,
      status,
      progress:
        status === "completed"
          ? 100
          : status === "locked" || need === 0
            ? 0
            : Math.round((done / need) * 100),
      found_clue_count: Math.min(found, def.required_clue_count),
    };
  });
}

// Seed one ready-to-play mission so every screen has content immediately.
const seedId = "mock-mission-1";
state.missions.set(
  seedId,
  buildMissionBundle(seedId, state.user.id, "detective", "medium", "Tehran", "en"),
);

function charge(action: string, missionId?: string): { coins_charged: number } {
  const price = MOCK_PRICING[action] ?? 0;
  if (state.wallet.balance < price) {
    throw new MockApiError(
      "insufficient_balance",
      "not enough coins",
      409,
    );
  }
  state.wallet.balance -= price;
  state.wallet.updated_at = now();
  state.transactions.unshift({
    id: mockId(),
    wallet_id: state.wallet.id,
    user_id: state.user.id,
    mission_id: missionId ?? null,
    type: action,
    amount: -price,
    balance_after: state.wallet.balance,
    metadata: {},
    created_at: now(),
  });
  return { coins_charged: price };
}

function credit(type: string, amount: number) {
  state.wallet.balance += amount;
  state.wallet.updated_at = now();
  state.transactions.unshift({
    id: mockId(),
    wallet_id: state.wallet.id,
    user_id: state.user.id,
    mission_id: null,
    type,
    amount,
    balance_after: state.wallet.balance,
    metadata: {},
    created_at: now(),
  });
  return state.transactions[0];
}

function tokens() {
  const in15m = new Date(Date.now() + 15 * 60_000).toISOString();
  const in30d = new Date(Date.now() + 30 * 86_400_000).toISOString();
  return {
    access_token: "mock-access-token",
    access_expires_at: in15m,
    refresh_token: "mock-refresh-token",
    refresh_expires_at: in30d,
  };
}

function bundleOr404(missionId: string): MockMissionBundle {
  const bundle = state.missions.get(missionId);
  if (!bundle) throw new MockApiError("not_found", "mission not found", 404);
  // Resolve pending generation.
  const until = state.generatingUntil.get(missionId);
  if (until !== undefined) {
    if (Date.now() >= until) {
      state.generatingUntil.delete(missionId);
      bundle.mission.status = "active";
      bundle.events.push({
        id: mockId(),
        mission_id: missionId,
        type: "generation_completed",
        payload: {},
        created_at: now(),
      });
    } else {
      bundle.mission.status = "generating";
    }
  }
  return bundle;
}

function delay(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}

const fillerFacts = [
  "The Thursday timeline is tighter than it first appeared.",
  "Someone else was interested in the same estate inventory.",
  "The chain on the warehouse was bought this week.",
];

export async function mockRequest<T>(
  path: string,
  opts: RequestOptions,
): Promise<T> {
  await delay(200 + Math.random() * 250);
  const method = opts.method ?? "GET";
  const body = (opts.body ?? {}) as Record<string, unknown>;
  const url = new URL(path, "http://mock.local");
  const p = url.pathname;
  const out = (v: unknown) => v as T;

  // ---- Auth ----
  if (p === "/api/v1/auth/register" && method === "POST") {
    state.user = {
      ...state.user,
      email: String(body.email ?? state.user.email),
      display_name: String(body.display_name ?? state.user.display_name),
    };
    return out({ user: state.user, tokens: tokens() });
  }
  if (p === "/api/v1/auth/login" && method === "POST") {
    state.user = { ...state.user, email: String(body.email ?? state.user.email) };
    return out({ user: state.user, tokens: tokens() });
  }
  if (p === "/api/v1/auth/refresh" && method === "POST") {
    return out({ tokens: tokens() });
  }
  if (p === "/api/v1/auth/logout" && method === "POST") {
    return out({ status: "logged_out" });
  }
  if (p === "/api/v1/me") {
    return out({ user: state.user });
  }

  // ---- Profile ----
  if (p === "/api/v1/profile" && method === "GET") {
    return out({ profile: mockProfile() });
  }
  if (p === "/api/v1/profile" && method === "PUT") {
    state.user.display_name = String(body.display_name ?? state.user.display_name);
    return out({ profile: mockProfile() });
  }
  if (p === "/api/v1/profile/stats") {
    return out({
      profile: mockProfile(),
      total_coins_spent: state.transactions
        .filter((t) => t.amount < 0)
        .reduce((s, t) => s - t.amount, 120),
      total_coins_earned: state.transactions
        .filter((t) => t.amount > 0)
        .reduce((s, t) => s + t.amount, 620),
    });
  }
  if (p === "/api/v1/profile/history") {
    return out({
      history: [...state.missions.values()].map((b) => ({
        mission_id: b.mission.id,
        title: b.mission.title,
        type: b.mission.type,
        difficulty: b.mission.difficulty,
        status: b.mission.status,
        created_at: b.mission.created_at,
        completed_at: b.mission.completed_at ?? null,
      })),
    });
  }
  if (p === "/api/v1/profile/badges") {
    return out({
      badges: [
        { id: "first-mission", name: "First Deployment" },
        { id: "clue-hound", name: "Clue Hound" },
      ],
    });
  }

  // ---- Wallet ----
  // Return copies so callers get snapshots, like a real API response.
  if (p === "/api/v1/wallet" && method === "GET") {
    return out({ wallet: { ...state.wallet } });
  }
  if (p === "/api/v1/wallet/transactions") {
    return out({ transactions: state.transactions.map((t) => ({ ...t })) });
  }
  if (p === "/api/v1/wallet/pricing") {
    return out({ pricing: MOCK_PRICING });
  }
  if (p === "/api/v1/wallet/config") {
    return out({
      demo_purchases: true,
      coin_packs: [
        { product_id: "coins_small", coins: 200 },
        { product_id: "coins_medium", coins: 600 },
        { product_id: "coins_large", coins: 1500 },
      ],
      rewarded_ad_coins: 20,
    });
  }
  if (p === "/api/v1/wallet/rewarded-ad/claim" && method === "POST") {
    if (state.adClaimedToday) {
      throw new MockApiError("ad_limit_reached", "daily ad limit reached", 409);
    }
    state.adClaimedToday = true;
    return out({ transaction: credit("rewarded_ad", 20) });
  }
  if (p === "/api/v1/wallet/purchase/verify" && method === "POST") {
    const amounts: Record<string, number> = {
      coins_small: 100,
      coins_medium: 550,
      coins_large: 1200,
    };
    return out({
      transaction: credit("purchase", amounts[String(body.product_id)] ?? 100),
    });
  }

  // ---- Missions ----
  if (p === "/api/v1/missions" && method === "POST") {
    charge("mission_start");
    const id = mockId();
    const bundle = buildMissionBundle(
      id,
      state.user.id,
      (body.type as Mission["type"]) ?? "detective",
      (body.difficulty as Mission["difficulty"]) ?? "medium",
      String(body.region ?? ""),
      body.language === "fa" ? "fa" : "en",
    );
    bundle.mission.status = "generating";
    bundle.events = [
      {
        id: mockId(),
        mission_id: id,
        type: "generation_started",
        payload: {},
        created_at: now(),
      },
    ];
    state.missions.set(id, bundle);
    state.generatingUntil.set(id, Date.now() + 8000);
    return out({ mission: bundle.mission });
  }
  if (p === "/api/v1/missions" && method === "GET") {
    return out({
      missions: [...state.missions.values()]
        .map((b) => bundleOr404(b.mission.id).mission)
        .sort((a, b) => (a.created_at < b.created_at ? 1 : -1)),
    });
  }

  const m = p.match(/^\/api\/v1\/missions\/([^/]+)(\/.*)?$/);
  if (m) {
    const missionId = m[1];
    const rest = m[2] ?? "";
    const bundle = bundleOr404(missionId);
    const generating = bundle.mission.status === "generating";
    const guardGenerating = () => {
      if (generating) {
        throw new MockApiError(
          "mission_generating",
          "mission is still generating",
          409,
        );
      }
    };

    if ((rest === "" || rest === "/dashboard") && method === "GET") {
      const objectives = Array.isArray(bundle.mission.objectives)
        ? bundle.mission.objectives
        : [];
      const completed = objectives.filter((o) => o.status === "completed");
      const active = objectives.filter((o) => o.status !== "completed");
      const progress =
        objectives.length > 0
          ? Math.round(
              objectives.reduce((sum, o) => sum + (o.progress ?? 0), 0) /
                objectives.length,
            )
          : 0;
      return out({
        mission_id: missionId,
        title: bundle.mission.title,
        mission_status: bundle.mission.status,
        mission: bundle.mission,
        primary_objective:
          objectives.find((o) => o.type === "primary") ?? objectives[0] ?? null,
        objectives: active,
        completed_objectives: completed,
        mission_progress: progress,
        risk_score: Number(bundle.mission.public_state?.risk_level ?? 22),
        current_time: bundle.mission.current_time,
        time_remaining: "18 hours",
        has_deadline: true,
        next_recommended_actions: [
          {
            type: "visit_location",
            title: "Check the recommended marker",
            description: "The next visible lead is most likely to move the case forward.",
            target_type: "location",
            target_id: bundle.markers.find((mk) => mk.recommended)?.id,
            priority: "high",
            cost_hint: "free",
          },
        ],
        guidance: {
          summary: "Follow the timeline and resolve conflicting timestamps.",
          warning: "Do not wait too long; risk rises as time advances.",
        },
        win_conditions: Array.isArray(bundle.mission.public_state?.win_conditions)
          ? (bundle.mission.public_state.win_conditions as string[])
          : [],
        failure_conditions: Array.isArray(
          bundle.mission.public_state?.failure_conditions,
        )
          ? (bundle.mission.public_state.failure_conditions as string[])
          : [],
        can_complete: progress >= 80,
        missing_requirements:
          progress >= 80 ? [] : ["Find more required clues before the final decision"],
        characters: generating ? [] : bundle.characters,
        clues: generating ? [] : bundle.clues,
        locations: generating ? [] : bundle.markers,
        timeline_preview: [...bundle.events].reverse().slice(0, 5),
        wallet_balance: state.wallet.balance,
        world_state: deriveMockWorldState(bundle),
        result: bundle.mission.result,
      });
    }
    if (rest === "/gameplay-status" && method === "GET") {
      const stages = mockStages(bundle);
      const current = stages.find((s) => s.status !== "completed") ?? null;
      const identify = stages.find((s) => s.unlock_on_complete.reveal_suspect);
      const suspectRevealed = identify?.status === "completed";
      const suspect = suspectRevealed
        ? (bundle.characters.find((c) => c.category === "antagonist") ??
          bundle.characters.find((c) => c.category !== "guide") ??
          null)
        : null;
      const pendingReport = current?.required_actions.find(
        (a) =>
          (a.type === "submit_report" || a.type === "final_decision") &&
          a.done < a.count,
      );
      const dashboard = await mockRequest<Record<string, unknown>>(
        `/api/v1/missions/${missionId}/dashboard`,
        { method: "GET" },
      );
      return out({
        ...dashboard,
        stages,
        current_stage: current,
        stage_index: current ? stages.indexOf(current) + 1 : 0,
        stage_count: stages.length,
        clue_goal: 5,
        clues_found: bundle.clues.length,
        suspect_status: suspectRevealed ? "identified" : "hidden",
        suspect,
        next_reward: current?.reward ?? null,
        report_pending: !!pendingReport,
        pending_report_type:
          pendingReport?.type === "final_decision"
            ? "final_report"
            : pendingReport?.report,
        board_art: {},
        stage_update: null,
      });
    }
    if (rest === "/actions/preview" && method === "POST") {
      const body = (opts.body ?? {}) as { action?: string };
      const minutes = MOCK_TIME_COSTS[body.action ?? ""];
      if (!minutes) {
        throw new MockApiError("invalid_action", "unknown action", 400);
      }
      return out({
        action: body.action,
        time_cost_minutes: minutes,
        coin_cost: MOCK_PRICING[body.action === "location_action" ? "location_search" : body.action ?? ""] ?? 0,
        risk_note: "",
        new_time_if_done: bundle.mission.current_time,
      });
    }
    if (rest === "/reports" && method === "GET") {
      return out({ reports: [...(mockReports.get(missionId) ?? [])].reverse() });
    }
    if (rest === "/reports" && method === "POST") {
      guardGenerating();
      const body = (opts.body ?? {}) as {
        type?: ReportType;
        title?: string;
        summary?: string;
        linked_clue_ids?: string[];
        suspect_character_id?: string;
      };
      const linked = body.linked_clue_ids ?? [];
      const confirmedLinked = linked.filter((id) =>
        bundle.clues.some((c) => c.id === id && c.status === "confirmed"),
      ).length;
      let verdict: "accepted" | "rejected" = "accepted";
      let feedback = "Report filed.";
      const missing: string[] = [];
      if (body.type === "clue_report" && linked.length === 0) {
        verdict = "rejected";
        missing.push("Link at least 1 discovered clue to the report");
        feedback = "A clue report must document actual evidence.";
      }
      if (body.type === "suspect_report") {
        if (!body.suspect_character_id) missing.push("Name a suspect character");
        if (confirmedLinked < 2)
          missing.push(`Link ${2 - confirmedLinked} more confirmed piece(s) of evidence`);
        if (missing.length > 0) {
          verdict = "rejected";
          feedback = "A suspect report needs a named suspect backed by confirmed evidence.";
        } else {
          feedback = "Suspect report accepted. Your named suspect is now the official line of investigation.";
        }
      }
      const report: MissionReport = {
        id: mockId(),
        mission_id: missionId,
        type: body.type ?? "progress_report",
        title: body.title ?? "",
        summary: body.summary ?? "",
        linked_clue_ids: linked,
        suspect_character_id: body.suspect_character_id ?? null,
        verdict,
        feedback,
        created_at: now(),
      };
      mockReports.set(missionId, [...(mockReports.get(missionId) ?? []), report]);
      bundle.events.push({
        id: mockId(),
        mission_id: missionId,
        type: verdict === "accepted" ? "report_accepted" : "report_rejected",
        payload: { type: report.type },
        created_at: now(),
      });
      return out({
        verdict,
        feedback,
        missing_requirements: missing,
        report,
        stage_update: null,
        time_update: {
          new_time: bundle.mission.current_time,
          minutes_advanced: 15,
          triggered_events: [],
        },
        timeline_events: ["report_submitted"],
        next_recommended_actions: [],
      });
    }
    if (rest === "/art/board/generate" && method === "POST") {
      const body = (opts.body ?? {}) as { board_type?: string };
      return out({
        board: {
          board_type: body.board_type ?? "mission_board_background",
          url: MOCK_PORTRAIT,
          status: "ready",
          version: 1,
        },
      });
    }
    if (rest === "/archive" && method === "POST") {
      bundle.mission.status = "archived";
      return out({ mission: bundle.mission });
    }
    if (rest === "/events") {
      return out({ events: [...bundle.events].reverse() });
    }
    if (rest === "/timeline") {
      // Curated player-facing timeline mirroring GET /missions/{id}/timeline.
      const typeMap: Record<string, string> = {
        mission_generated: "mission_started",
        mission_ready: "mission_started",
        location_discovered: "new_location_unlocked",
        location_visited: "location_visited",
        location_action: "location_visited",
        clue_discovered: "clue_discovered",
        evidence_confirmed: "evidence_confirmed",
        location_unlocked: "new_location_unlocked",
        hypothesis_submitted: "hypothesis_submitted",
        dialogue: "character_talked",
        character_chat: "character_talked",
        ai_guidance: "ai_guidance_received",
        time_advanced: "time_advanced",
        mission_completed: "mission_completed",
        mission_failed: "mission_failed",
      };
      const items = bundle.events
        .map((ev) => {
          const type = typeMap[ev.type];
          if (!type) return null;
          const payload = ev.payload as Record<string, unknown>;
          const title = String(
            payload.title ?? payload.name ?? payload.location ?? "",
          );
          if (!title && type !== "mission_completed") return null;
          return {
            id: ev.id,
            type,
            title: title || bundle.mission.title,
            occurred_at: ev.created_at,
            importance:
              type === "clue_discovered" ||
              type === "mission_started" ||
              type === "mission_completed" ||
              type === "mission_failed"
                ? "high"
                : "medium",
          };
        })
        .filter(Boolean);
      return out({ mission_id: missionId, items });
    }
    if (rest === "/completion-check") {
      const objectives = Array.isArray(bundle.mission.objectives)
        ? bundle.mission.objectives
        : [];
      const progress =
        objectives.length > 0
          ? Math.round(
              objectives.reduce((sum, o) => sum + (o.progress ?? 0), 0) /
                objectives.length,
            )
          : 0;
      const missing =
        progress >= 80 ? [] : ["Find more required clues before the final decision"];
      return out({
        can_complete: progress >= 80,
        reason:
          progress >= 80
            ? "The mission is ready for your final decision."
            : missing[0],
        missing_requirements: missing,
      });
    }
    if (rest === "/complete" && method === "POST") {
      const cost = charge("final_judgment", missionId);
      bundle.mission.status = "completed";
      bundle.mission.completed_at = now();
      const result = {
        can_complete: true,
        success: true,
        score: 84,
        stars: 4,
        result_title: "Mission Successful",
        result_summary:
          "Your final call fits the visible evidence and resolves the core objective.",
        completed_objectives: ["Resolve the core timeline"],
        failed_objectives: [],
        missed_optional_objectives: ["Recover every minor witness detail"],
        critical_clues_found: bundle.clues.slice(0, 2).map((c) => c.title),
        critical_clues_missed: [],
        good_decisions: ["You connected the warehouse chain to the loan slip."],
        bad_decisions: [],
        xp_reward: 420,
        coin_reward: 40,
        cost,
      };
      bundle.mission.result = result;
      bundle.events.push({
        id: mockId(),
        mission_id: missionId,
        type: "mission_completed",
        payload: { title: result.result_title },
        created_at: now(),
      });
      return out(result);
    }
    if (rest === "/result" && method === "GET") {
      if (!bundle.mission.result) {
        throw new MockApiError(
          "mission_not_finished",
          "this mission has no result yet",
          409,
        );
      }
      return out({
        mission_id: missionId,
        mission_status: bundle.mission.status,
        result: bundle.mission.result,
      });
    }
    if (rest === "/map") {
      guardGenerating();
      return out({
        mission_id: missionId,
        center: { lat: bundle.mission.center_lat, lng: bundle.mission.center_lng },
        zoom: bundle.mission.map_zoom,
        locations: bundle.markers,
      });
    }

    const loc = rest.match(/^\/locations\/([^/]+)(\/.*)?$/);
    if (loc) {
      guardGenerating();
      const marker = bundle.markers.find((mk) => mk.id === loc[1]);
      if (!marker) throw new MockApiError("not_found", "location not found", 404);
      const desc = bundle.locationDescriptions[marker.id];
      if ((loc[2] ?? "") === "" && method === "GET") {
        return out({
          location: {
            id: marker.id,
            mission_id: missionId,
            name: marker.name,
            type: marker.type,
            latitude: marker.lat,
            longitude: marker.lng,
            status: marker.status,
            risk_level: marker.risk_level,
            description: desc?.description ?? "",
            available_actions: desc?.actions ?? ["inspect_area"],
            created_at: now(),
            updated_at: now(),
          },
          characters: bundle.characters.filter(
            (c) => c.current_location_id === marker.id,
          ),
          discovered_clues: bundle.clues.filter(
            (c) => c.location_id === marker.id,
          ),
        });
      }
      if (loc[2] === "/actions" && method === "POST") {
        const cost = charge("location_search", missionId);
        marker.status = "visited";
        marker.has_new_clue = false;
        bundle.events.push({
          id: mockId(),
          mission_id: missionId,
          type: "location_action",
          payload: { location: marker.name, action: body.action },
          created_at: now(),
        });
        return out({
          narrative:
            `You ${String(body.action ?? "inspect").replace(/_/g, " ")} at ${marker.name}. ` +
            "Details emerge that were easy to miss on first glance.",
          discovered_clues: [],
          new_facts: [fillerFacts[Math.floor(Math.random() * fillerFacts.length)]],
          cost,
        });
      }
      if (loc[2] === "/ask-ai" && method === "POST") {
        const cost = charge("location_ask", missionId);
        return out({
          message:
            `Focus on what ${marker.name} can still tell you. ` +
            "Cross-check timestamps before trusting testimony.",
          hint_level: "gentle",
          referenced_items: [
            { type: "location", id: marker.id, name: marker.name },
          ],
          cost,
        });
      }
    }

    const ch = rest.match(/^\/characters(?:\/([^/]+))?(\/chat|\/avatar)?$/);
    if (ch) {
      guardGenerating();
      if (!ch[1]) {
        return out({ characters: bundle.characters });
      }
      const character = bundle.characters.find((c) => c.id === ch[1]);
      if (!character)
        throw new MockApiError("not_found", "character not found", 404);
      if (!ch[2]) {
        return out({ character, messages: [] });
      }
      if (ch[2] === "/avatar" && method === "POST") {
        character.avatar_url = MOCK_PORTRAIT;
        character.avatar_status = "ready";
        character.avatar_version = (character.avatar_version ?? 0) + 1;
        return out({ character });
      }
      if (method === "POST") {
        const cost = charge("character_chat", missionId);
        const delta = Math.random() > 0.5 ? 3 : -2;
        character.trust_level = Math.max(
          0,
          Math.min(100, character.trust_level + delta),
        );
        bundle.events.push({
          id: mockId(),
          mission_id: missionId,
          type: "character_chat",
          payload: { character: character.name },
          created_at: now(),
        });
        return out({
          message:
            `"${String(body.message ?? "")}" — ` +
            `${character.name} pauses before answering, choosing words carefully. ` +
            "They mention Thursday evening without being asked about it.",
          emotion: "guarded",
          mood: character.mood,
          trust_level: character.trust_level,
          trust_delta: delta,
          stress_delta: delta < 0 ? 2 : -1,
          unlocked_clues: [],
          new_facts: [fillerFacts[Math.floor(Math.random() * fillerFacts.length)]],
          cost,
        });
      }
    }

    if (rest === "/hypothesis" && method === "POST") {
      const confirmedCount = bundle.clues.filter(
        (c) => c.status === "confirmed",
      ).length;
      const linked = Array.isArray(body.clue_ids) ? body.clue_ids : [];
      const linkedConfirmed = bundle.clues.filter(
        (c) => c.status === "confirmed" && linked.includes(c.id),
      ).length;
      let verdict: "too_early" | "unsupported" | "partially_correct";
      let feedback: string;
      let needsMore = true;
      if (confirmedCount < 2) {
        verdict = "too_early";
        feedback =
          "It is too early to commit to a theory — confirm more evidence first.";
      } else if (linkedConfirmed === 0) {
        verdict = "unsupported";
        feedback =
          "Your theory is not yet backed by confirmed evidence. Link confirmed clues.";
      } else {
        verdict = "partially_correct";
        needsMore = false;
        feedback =
          "Your theory is consistent with your confirmed evidence. Submit your final decision when ready.";
      }
      bundle.events.push({
        id: mockId(),
        mission_id: missionId,
        type: "hypothesis_submitted",
        payload: { verdict, summary: String(body.answer ?? "").slice(0, 80) },
        created_at: now(),
      });
      return out({
        verdict,
        feedback,
        needs_more_evidence: needsMore,
        confirmed_count: confirmedCount,
        progression: {
          message: feedback,
          state_changes: [],
          timeline_events: ["hypothesis_submitted"],
          unlocked_locations: [],
          next_recommended_actions:
            verdict === "partially_correct"
              ? [{ type: "prepare_final", title: "Prepare your final decision" }]
              : [],
        },
      });
    }

    const cl = rest.match(/^\/clues(?:\/([^/]+))?(\/inspect|\/explain|\/image|\/confirm)?$/);
    if (cl) {
      guardGenerating();
      if (!cl[1]) {
        return out({ clues: bundle.clues.filter((c) => c.discovered) });
      }
      const clue = bundle.clues.find((c) => c.id === cl[1]);
      if (!clue) throw new MockApiError("not_found", "clue not found", 404);
      if (!cl[2]) {
        return out({ clue });
      }
      if (cl[2] === "/image" && method === "POST") {
        clue.image_url = MOCK_PORTRAIT;
        clue.image_status = "ready";
        clue.image_version = (clue.image_version ?? 0) + 1;
        return out({ clue });
      }
      if (cl[2] === "/confirm" && method === "POST") {
        const already = clue.status === "confirmed";
        clue.status = "confirmed";
        // Unlock the earliest still-locked marker on first confirm.
        const unlocked: Array<{ id: string; name: string; reason: string }> = [];
        if (!already) {
          const locked = bundle.markers.find((m) => m.is_locked);
          if (locked) {
            locked.is_locked = false;
            locked.status = "discovered";
            unlocked.push({
              id: locked.id,
              name: locked.name,
              reason: `Unlocked after confirming evidence: ${clue.title}`,
            });
            bundle.events.push({
              id: mockId(),
              mission_id: missionId,
              type: "location_unlocked",
              payload: { name: locked.name, reason: unlocked[0].reason },
              created_at: now(),
            });
          }
          bundle.events.push({
            id: mockId(),
            mission_id: missionId,
            type: "evidence_confirmed",
            payload: { title: clue.title, clue_id: clue.id },
            created_at: now(),
          });
        }
        return out({
          message: already ? "This evidence is already confirmed." : "Evidence confirmed.",
          state_changes: already
            ? []
            : [{ entity: "clue", id: clue.id, from: "discovered", to: "confirmed" }],
          timeline_events: already ? [] : ["evidence_confirmed"],
          unlocked_locations: unlocked,
          next_recommended_actions: [],
        });
      }
      if (cl[2] === "/inspect" && method === "POST") {
        const cost = charge("clue_inspect", missionId);
        clue.reliability = Math.min(100, clue.reliability + 5);
        return out({
          analysis:
            `Close inspection of "${clue.title}" sharpens the picture: ` +
            "the physical details are consistent with the Thursday timeline.",
          new_facts: [fillerFacts[0]],
          clue,
          cost,
        });
      }
      if (cl[2] === "/explain" && method === "POST") {
        const cost = charge("clue_explain", missionId);
        return out({
          explanation:
            `"${clue.title}" matters because it anchors a moment in time ` +
            "that two people describe differently.",
          next_steps: [
            "Compare it with testimony from the café.",
            "Verify the stamped time against the loan slip.",
          ],
          compare_with: bundle.clues
            .filter((c) => c.id !== clue.id)
            .slice(0, 2)
            .map((c) => c.title),
          cost,
        });
      }
    }

    if (rest === "/guidance" && method === "POST") {
      guardGenerating();
      const cost = charge("ai_guidance", missionId);
      return out({
        message:
          "Work the timeline. You hold three timestamps that do not agree — " +
          "resolving that disagreement is your next move.",
        hint_level: "moderate",
        referenced_items: bundle.clues.slice(0, 2).map((c) => ({
          type: "clue",
          id: c.id,
          name: c.title,
        })),
        cost,
      });
    }

    if (rest === "/time" && method === "GET") {
      guardGenerating();
      return out({
        current_time: bundle.mission.current_time,
        public_state: bundle.mission.public_state,
      });
    }
    if (rest === "/time/advance" && method === "POST") {
      guardGenerating();
      const cost = charge("advance_time", missionId);
      const amount = Number(body.amount ?? 1);
      const unit = String(body.unit ?? "hours");
      bundle.mission.current_time = `Day 1 — ${9 + (amount % 12)}:00`;
      bundle.events.push({
        id: mockId(),
        mission_id: missionId,
        type: "time_advanced",
        payload: { amount, unit },
        created_at: now(),
      });
      return out({
        new_time: bundle.mission.current_time,
        summary: `Time moves ${amount} ${unit} forward. The city keeps its secrets a little less tightly.`,
        events: [{ type: "world", title: "Rain stops over the archive district" }],
        cost,
      });
    }

    const jr = rest.match(/^\/journal(?:\/([^/]+))?$/);
    if (jr) {
      if (!jr[1]) {
        if (method === "GET") return out({ notes: bundle.notes });
        if (method === "POST") {
          const note = {
            id: mockId(),
            mission_id: missionId,
            title: String(body.title ?? ""),
            content: String(body.content ?? ""),
            created_at: now(),
            updated_at: now(),
          };
          bundle.notes.unshift(note);
          return out({ note });
        }
      } else {
        const note = bundle.notes.find((n) => n.id === jr[1]);
        if (!note) throw new MockApiError("not_found", "note not found", 404);
        if (method === "PUT") {
          note.title = String(body.title ?? note.title);
          note.content = String(body.content ?? note.content);
          note.updated_at = now();
          return out({ note });
        }
        if (method === "DELETE") {
          bundle.notes = bundle.notes.filter((n) => n.id !== jr[1]);
          return out(undefined);
        }
      }
    }
  }

  throw new MockApiError("not_found", `no mock for ${method} ${p}`, 404);
}

function mockProfile() {
  return {
    id: "mock-profile-1",
    user_id: state.user.id,
    display_name: state.user.display_name,
    rank: "Field Agent",
    level: 7,
    xp: 3420,
    total_missions: state.missions.size + 11,
    completed_missions: 9,
    failed_missions: 2,
    success_rate: 0.75,
    favorite_mission_type: "detective",
    total_clues_found: 47,
    total_ai_interactions: 129,
    total_locations_visited: 58,
    badges: [
      { id: "first-mission", name: "First Deployment" },
      { id: "clue-hound", name: "Clue Hound" },
    ],
    created_at: now(),
    updated_at: now(),
  };
}
