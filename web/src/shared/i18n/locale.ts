import type { Language } from "@/shared/types/api";

/**
 * Framework-agnostic locale state. The I18nProvider keeps this in sync via
 * setActiveLanguage() so the plain (non-React) API client can read the current
 * language when building request headers/bodies. Falls back to the DOM/storage
 * so it is correct even before the provider mounts.
 */

let active: Language | null = null;

const STORAGE_KEY = "agentverse.lang";

export function setActiveLanguage(lang: Language) {
  active = lang;
}

export function currentLanguage(): Language {
  if (active) return active;
  try {
    const docLang = document.documentElement.lang;
    if (docLang === "fa" || docLang === "en") return docLang;
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "fa" || stored === "en") return stored;
  } catch {
    // ignore
  }
  return "en";
}

export function currentDirection(): "rtl" | "ltr" {
  return currentLanguage() === "fa" ? "rtl" : "ltr";
}

/** BCP-47 tag for Accept-Language / locale fields. */
export function currentLocaleTag(): string {
  return currentLanguage() === "fa" ? "fa-IR" : "en-US";
}

/** The response contract embedded in AI request bodies. */
export function responseContract() {
  const language = currentLanguage();
  return {
    language,
    direction: currentDirection(),
    style: "immersive_game_ui",
    do_not_switch_language: true,
  } as const;
}

/**
 * Augment an AI request body with language metadata so the backend produces a
 * response in the selected UI language.
 */
export function withLocale<T extends Record<string, unknown>>(payload: T) {
  return {
    ...payload,
    locale: currentLocaleTag(),
    language: currentLanguage(),
    ui_direction: currentDirection(),
    response_language: currentLanguage(),
    response_contract: responseContract(),
  };
}

/** Locale headers attached to every request. */
export function localeHeaders(): Record<string, string> {
  return {
    "Accept-Language": currentLocaleTag(),
    "X-App-Language": currentLanguage(),
    "X-UI-Direction": currentDirection(),
  };
}

/**
 * Heuristic language check for the dev-only mismatch guard: does `text` look
 * like it is written in the expected language? Persian text contains Arabic-
 * script codepoints; English/Latin text does not.
 */
export function looksLikeLanguage(text: string, lang: Language): boolean {
  const persian = /[؀-ۿ]/;
  const hasPersian = persian.test(text);
  return lang === "fa" ? hasPersian : !hasPersian || /[a-zA-Z]/.test(text);
}
