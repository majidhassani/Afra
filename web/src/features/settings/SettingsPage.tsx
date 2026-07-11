import { useNavigate } from "react-router-dom";
import { LogOut, RotateCcw } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { useAuthStore, getRefreshToken } from "@/features/auth/authStore";
import { authApi } from "@/shared/api/endpoints";
import { Button } from "@/shared/ui/Button";
import { replayOnboarding } from "@/shared/ui/Onboarding";
import { toast } from "@/shared/ui/toast";

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

      {/* Ordinary preferences — calm, plain glass, no gameplay chrome. */}
      <div className="band-title" style={{ marginBottom: 8 }}>
        {t("settings.preferences")}
      </div>
      <div className="panel" style={{ marginBottom: 22 }}>
        <div className="band spread">
          <div>
            <h3>{t("settings.language")}</h3>
            <p className="faint">{t("settings.language.body")}</p>
          </div>
          <LanguageSwitcher />
        </div>
        <div className="band spread" style={{ borderBottom: "none" }}>
          <div>
            <h3>{t("onboarding.replay")}</h3>
            <p className="faint">{t("onboarding.welcome.title")}</p>
          </div>
          <Button
            variant="secondary"
            onClick={() => {
              replayOnboarding();
              toast("success", t("onboarding.replayed"));
            }}
          >
            <RotateCcw size={14} aria-hidden />
            {t("onboarding.replay")}
          </Button>
        </div>
      </div>

      {/* Account / destructive actions — deliberately set apart visually
          (a semantic danger border, not the ordinary panel treatment) so
          "sign out" never reads as just another preference row. */}
      <div className="band-title" style={{ marginBottom: 8 }}>
        {t("settings.account")}
      </div>
      <div className="glass-3 glass-3--danger" style={{ padding: 0 }}>
        <div className="band spread" style={{ borderBottom: "none" }}>
          <div>
            <h3>{t("settings.session")}</h3>
            <p className="faint">
              {user?.email} — {t("settings.account.body")} {t("settings.logout.body")}
            </p>
          </div>
          <Button variant="danger" onClick={logout}>
            <LogOut size={14} aria-hidden />
            {t("nav.logout")}
          </Button>
        </div>
      </div>
    </div>
  );
}
