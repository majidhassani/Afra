import { Link } from "react-router-dom";
import { useI18n } from "@/shared/i18n";

export function NotFoundPage() {
  const { t } = useI18n();
  return (
    <div className="state-box" style={{ minHeight: "60dvh" }}>
      <div className="state-title">{t("common.notFound")}</div>
      <p className="faint">{t("common.notFound.body")}</p>
      <Link className="btn btn-secondary" to="/app/dashboard">
        {t("nav.dashboard")}
      </Link>
    </div>
  );
}
