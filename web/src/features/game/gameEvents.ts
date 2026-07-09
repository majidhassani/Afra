/**
 * Game event bus: gameplay responses (stage completions, rewards, suspect
 * reveal, world events, time costs) push typed events here, and the
 * GameEffectsLayer renders them as Unity-style popups wherever the player is.
 */
import { create } from "zustand";
import type { PublicCharacter, Stage, StageUpdate, TimeUpdate } from "@/shared/types/api";

export type GameEvent =
  | { kind: "stage_complete"; id: number; stage: Stage }
  | { kind: "stage_unlocked"; id: number; stage: Stage }
  | { kind: "reward"; id: number; xp: number; coins: number; badge?: string }
  | { kind: "suspect"; id: number; suspect: PublicCharacter | null }
  | { kind: "location_unlocked"; id: number; name: string; reason: string }
  | { kind: "world_event"; id: number; title: string; description?: string }
  | { kind: "time"; id: number; minutes: number; newTime: string };

/** Omit distributed over a union, so each event variant keeps its fields. */
type DistributiveOmit<T, K extends PropertyKey> = T extends unknown
  ? Omit<T, K>
  : never;

interface GameEventState {
  queue: GameEvent[];
  push: (ev: DistributiveOmit<GameEvent, "id">) => void;
  dismiss: (id: number) => void;
}

let nextId = 1;

export const useGameEvents = create<GameEventState>((set) => ({
  queue: [],
  push: (ev) =>
    set((s) => ({ queue: [...s.queue, { ...ev, id: nextId++ } as GameEvent] })),
  dismiss: (id) => set((s) => ({ queue: s.queue.filter((e) => e.id !== id) })),
}));

/** Fold a stage-engine update into popups (rewards, unlocks, reveal). */
export function announceStageUpdate(update: StageUpdate | null | undefined) {
  if (!update) return;
  const push = useGameEvents.getState().push;
  for (const stage of update.completed_stages ?? []) {
    push({ kind: "stage_complete", stage });
    const r = stage.reward;
    if (r && (r.xp > 0 || r.coins > 0 || r.badge)) {
      push({ kind: "reward", xp: r.xp, coins: r.coins, badge: r.badge });
    }
  }
  for (const loc of update.unlocked_locations ?? []) {
    push({ kind: "location_unlocked", name: loc.name, reason: loc.reason });
  }
  if (update.suspect_revealed) {
    push({ kind: "suspect", suspect: update.suspect ?? null });
  }
  if (update.activated_stage) {
    push({ kind: "stage_unlocked", stage: update.activated_stage });
  }
}

/** Fold a time update into a "time passed" toast + world event popups. */
export function announceTimeUpdate(update: TimeUpdate | null | undefined) {
  if (!update || update.minutes_advanced <= 0) return;
  const push = useGameEvents.getState().push;
  push({ kind: "time", minutes: update.minutes_advanced, newTime: update.new_time });
  for (const ev of update.triggered_events ?? []) {
    push({ kind: "world_event", title: ev.title, description: ev.description });
  }
}
