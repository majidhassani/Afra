import { useNavigate } from "react-router-dom";
import { LogOut } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { useAuthStore, getRefreshToken } from "@/features/auth/authStore";
import { authApi } from "@/shared/api/endpoints";

export function SettingsPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const clear = useAuthStore((s) => s.clear);
  const user = useAuthStore((s) => s.session?.user);

  const logout = async () => {
    const refreshToken = getRefreshToken();
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch {
      // Best-effort server logout.
    }
    clear();
    navigate("/login");
  };

  return (
    <div className="page" style={{ maxWidth: 720 }}>
      <header className="page-header">
        <h1>{t("settings.title")}</h1>
      </header>

      <div className="panel">
        <div className="band spread">
          <div>
            <h3>{t("settings.language")}</h3>
            <p className="faint">{t("settings.language.body")}</p>
          </div>
          <LanguageSwitcher />
        </div>
        <div className="band spread" style={{ borderBottom: "none" }}>
          <div>
            <h3>{t("settings.session")}</h3>
            <p className="faint">
              {user?.email} — {t("settings.logout.body")}
            </p>
          </div>
          <button className="btn btn-danger" onClick={logout}>
            <LogOut size={14} aria-hidden />
            {t("nav.logout")}
          </button>
        </div>
      </div>
    </div>
  );
}
