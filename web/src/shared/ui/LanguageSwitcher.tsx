import { useI18n } from "@/shared/i18n";

export function LanguageSwitcher() {
  const { lang, setLang, t } = useI18n();
  return (
    <div className="segmented" role="group" aria-label={t("settings.language")}>
      <button
        type="button"
        aria-pressed={lang === "en"}
        onClick={() => setLang("en")}
      >
        EN
      </button>
      <button
        type="button"
        aria-pressed={lang === "fa"}
        onClick={() => setLang("fa")}
      >
        فا
      </button>
    </div>
  );
}
