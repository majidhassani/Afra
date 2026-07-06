import type { Marker, Mission, Objective, PublicClue } from "@/shared/types/api";

/**
 * Deterministic, client-side mission HUD derivation.
 *
 * The backend now stores structured objectives (type/progress/status),
 * win/failure hints, risk_level and a deadline inside the mission and its
 * public_state. This computes a readable command-center HUD from that data,
 * degrading gracefully when older fields are missing so it works against both
 * the live API and the mock.
 */

export interface MissionHud {
  primary: Objective | null;
  progress: number;
  risk: number;
  riskBand: "low" | "med" | "high";
  winConditions: string[];
  recommended: Marker | null;
}

function num(v: unknown): number | null {
  if (typeof v === "number") return v;
  if (typeof v === "string" && v.trim() !== "" && !Number.isNaN(Number(v))) {
    return Number(v);
  }
  return null;
}

function strings(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : [];
}

export function deriveHud(
  mission: Mission,
  objectives: Objective[],
  clues: PublicClue[],
  locations: Marker[],
): MissionHud {
  const primary =
    objectives.find((o) => o.type === "primary") ??
    objectives.find((o) => o.status === "active" && !o.optional) ??
    objectives[0] ??
    null;

  // Progress: prefer stored objective progress; otherwise derive from
  // completed objectives + discovered clue coverage.
  const withProgress = objectives.filter((o) => typeof o.progress === "number");
  let progress: number;
  if (primary && typeof primary.progress === "number" && primary.progress > 0) {
    progress = primary.progress;
  } else if (withProgress.length > 0) {
    progress = Math.round(
      withProgress.reduce((s, o) => s + (o.progress ?? 0), 0) / withProgress.length,
    );
  } else {
    const mandatory = objectives.filter((o) => !o.optional);
    const completed = mandatory.filter((o) => o.status === "completed").length;
    const objRatio = mandatory.length ? completed / mandatory.length : 0;
    const need = objectives.reduce((s, o) => s + (o.required_clues || 0), 0);
    const clueRatio = need ? Math.min(1, clues.length / need) : 0;
    progress = Math.round(objRatio * 60 + clueRatio * 40);
  }
  progress = Math.max(0, Math.min(100, progress));

  const ps = (mission.public_state ?? {}) as Record<string, unknown>;
  const risk = Math.max(0, Math.min(100, num(ps.risk_level) ?? 0));
  const riskBand = risk >= 66 ? "high" : risk >= 33 ? "med" : "low";
  const winConditions = strings(ps.win_conditions);

  // Recommended next: a flagged marker, else the first accessible unvisited one.
  const recommended =
    locations.find((m) => m.recommended) ??
    locations.find((m) => !m.is_locked && m.status === "discovered") ??
    locations.find((m) => m.has_new_clue && !m.is_locked) ??
    null;

  return { primary, progress, risk, riskBand, winConditions, recommended };
}
