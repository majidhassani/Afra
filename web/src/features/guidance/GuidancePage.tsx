import { useParams } from "react-router-dom";
import { Radio } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { GuidancePanel } from "./GuidancePanel";

/**
 * GuidancePage — the full-screen "Mission Control" AI assistant, reachable as
 * the AI tab in the mobile bottom navigation. It wraps the shared
 * GuidancePanel so the same no-spoiler, locale-aware assistant is available
 * from a dedicated screen.
 */
export function GuidancePage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();

  return (
    <div className="page">
      <header className="page-header">
        <h1 className="row" style={{ gap: 10 }}>
          <Radio size={22} aria-hidden />
          {t("guidance.title")}
        </h1>
        <p className="subtitle">{t("guidance.subtitle")}</p>
      </header>
      <div className="panel" style={{ padding: 16 }}>
        {missionId && <GuidancePanel missionId={missionId} screen="ai" />}
      </div>
    </div>
  );
}
