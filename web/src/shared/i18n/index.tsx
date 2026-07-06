import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { en, type TranslationKey } from "./en";
import { fa } from "./fa";
import { env } from "@/shared/config/env";
import type { Language } from "@/shared/types/api";
import {
  setActiveLanguage,
  currentLocaleTag,
  responseContract,
} from "./locale";

const STORAGE_KEY = "agentverse.lang";

const dictionaries: Record<Language, Record<TranslationKey, string>> = {
  en,
  fa,
};

export type Translate = (
  key: TranslationKey,
  params?: Record<string, string | number>,
) => string;

interface I18nValue {
  lang: Language;
  dir: "ltr" | "rtl";
  setLang: (lang: Language) => void;
  t: Translate;
}

const I18nContext = createContext<I18nValue | null>(null);

function readStoredLang(): Language {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "en" || stored === "fa") return stored;
  } catch {
    // localStorage unavailable (private mode etc.) — fall through.
  }
  return env.defaultLanguage;
}

function applyDocumentLang(lang: Language) {
  document.documentElement.lang = lang;
  document.documentElement.dir = lang === "fa" ? "rtl" : "ltr";
  // Keep the framework-agnostic locale state in sync so the API client can
  // read the active language when building request headers/bodies.
  setActiveLanguage(lang);
}

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Language>(() => {
    const initial = readStoredLang();
    applyDocumentLang(initial);
    return initial;
  });

  const setLang = useCallback((next: Language) => {
    setLangState(next);
    applyDocumentLang(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // Ignore storage failures.
    }
  }, []);

  const t = useCallback<Translate>(
    (key, params) => {
      let text = dictionaries[lang][key] ?? en[key] ?? key;
      if (params) {
        for (const [name, value] of Object.entries(params)) {
          text = text.replaceAll(`{${name}}`, String(value));
        }
      }
      return text;
    },
    [lang],
  );

  const value = useMemo<I18nValue>(
    () => ({ lang, dir: lang === "fa" ? "rtl" : "ltr", setLang, t }),
    [lang, setLang, t],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside I18nProvider");
  return ctx;
}

/**
 * useLocale exposes the language metadata AI requests and the dev language
 * guard need, derived from the active i18n language.
 */
export function useLocale() {
  const { lang, dir } = useI18n();
  return {
    language: lang,
    direction: dir,
    locale: currentLocaleTag(),
    contract: responseContract(),
  };
}

export type { TranslationKey };
