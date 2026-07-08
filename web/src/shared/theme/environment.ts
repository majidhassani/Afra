import { useEffect } from "react";
import type { Mission, MissionType, WorldState } from "@/shared/types/api";

/**
 * Environment theming — the game UI recolours itself to match the mission's
 * world (jungle / city / desert / night), the way a AAA HUD tints to its
 * biome. Each environment drives the accent, glow, and cinematic backdrop via
 * the [data-env] attribute (see cinematic.css).
 */
export type Environment =
  | "forest"
  | "desert"
  | "city"
  | "snow"
  | "horror"
  | "space";

const BY_TYPE: Record<MissionType, Environment> = {
  wildlife_rescue: "forest",
  exploration: "space",
  survival: "snow",
  disaster_response: "desert",
  diplomacy: "city",
  detective: "city",
  medical_mystery: "horror",
};

// Region/summary keywords nudge the environment when the text is evocative.
// Checked in order — first match wins.
const REGION_HINTS: Array<[RegExp, Environment]> = [
  [/forest|jungle|rain\s?forest|amazon|woods|جنگل|بیشه|درخت/i, "forest"],
  [/desert|dune|sahara|arid|canyon|بیابان|کویر|صحرا/i, "desert"],
  [/snow|ice|arctic|glacier|frozen|tundra|blizzard|برف|یخ|قطب/i, "snow"],
  [/space|orbit|station|cosmic|nebula|mars|فضا|مدار|سیاره/i, "space"],
  [/horror|haunt|nightmare|asylum|morgue|وحشت|ترسناک|جن/i, "horror"],
  [/city|urban|street|downtown|metro|neon|شهر|خیابان|مترو/i, "city"],
];

/** Derive the environment for a mission from its region text, then its type. */
export function environmentFor(mission: Pick<Mission, "type" | "region" | "summary">): Environment {
  const text = `${mission.region ?? ""} ${mission.summary ?? ""}`;
  for (const [pattern, env] of REGION_HINTS) {
    if (pattern.test(text)) return env;
  }
  return BY_TYPE[mission.type] ?? "city";
}

/**
 * Applies [data-env] to the document root while a themed screen is mounted and
 * clears it on unmount, so the cinematic backdrop only tints in-mission.
 */
export function useEnvironmentTheme(env: Environment | null | undefined) {
  useEffect(() => {
    const root = document.documentElement;
    if (!env) {
      delete root.dataset.env;
      return;
    }
    root.dataset.env = env;
    return () => {
      delete root.dataset.env;
    };
  }, [env]);
}

/**
 * Applies the living-world modifiers (weather / time-of-day / danger) to the
 * document root so the shell can layer rain, night darkening, and a danger
 * pulse over the biome theme. Cleared when no world state is active.
 */
export function useWorldModifiers(world: WorldState | null | undefined) {
  const weather = world?.weather ?? null;
  const tod = world?.time_of_day ?? null;
  const danger = world?.danger ?? false;
  useEffect(() => {
    const root = document.documentElement;
    if (!world) {
      delete root.dataset.weather;
      delete root.dataset.tod;
      delete root.dataset.danger;
      return;
    }
    if (weather && weather !== "clear") root.dataset.weather = weather;
    else delete root.dataset.weather;
    if (tod) root.dataset.tod = tod;
    else delete root.dataset.tod;
    if (danger) root.dataset.danger = "true";
    else delete root.dataset.danger;
    return () => {
      delete root.dataset.weather;
      delete root.dataset.tod;
      delete root.dataset.danger;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [weather, tod, danger, !!world]);
}
