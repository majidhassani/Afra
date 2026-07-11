import type { ReactNode } from "react";
import { useI18n } from "@/shared/i18n";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";

export function AuthLayout({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="auth-layout">
      <aside className="auth-side">
        <div className="spread">
          <span className="auth-brand">
            <span className="auth-brand-mark" aria-hidden>
              {t("common.appName").charAt(0)}
            </span>
            {t("common.appName")}
          </span>
          <LanguageSwitcher />
        </div>
        <div className="auth-tagline">
          <h1>{t("auth.tagline")}</h1>
        </div>
        <p className="faint">© {new Date().getFullYear()} AgentVerse</p>
      </aside>
      <div className="auth-form-wrap">{children}</div>
    </div>
  );
}
