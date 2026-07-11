import { useI18n } from "@/shared/i18n";
import { Button } from "./Button";

export function NotFoundPage() {
  const { t } = useI18n();
  return (
    <div className="state-box" style={{ minHeight: "60dvh" }}>
      <div className="state-title">{t("common.notFound")}</div>
      <p className="faint">{t("common.notFound.body")}</p>
      <Button to="/app/dashboard" variant="secondary">
        {t("nav.dashboard")}
      </Button>
    </div>
  );
}
