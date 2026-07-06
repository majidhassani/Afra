import { useCallback } from "react";
import { useI18n } from "./index";
import { looksLikeLanguage } from "./locale";
import { toast } from "@/shared/ui/toast";

/**
 * Dev-only guard against AI responses that come back in the wrong language.
 * In production it is a no-op — we never block or annotate the game UI. In
 * development it surfaces a single non-blocking warning toast so language
 * regressions are caught early.
 */
export function useLanguageGuard() {
  const { lang } = useI18n();
  return useCallback(
    (text: string | undefined | null) => {
      if (!import.meta.env.DEV) return;
      if (!text || text.trim().length < 8) return;
      if (!looksLikeLanguage(text, lang)) {
        toast(
          "info",
          `Dev warning: AI response language does not match selected locale (${lang}).`,
        );
      }
    },
    [lang],
  );
}
