import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CheckCircle2, FileText, XCircle } from "lucide-react";
import { reportsApi } from "@/shared/api/endpoints";
import { useI18n } from "@/shared/i18n";
import { GameButton } from "@/shared/ui/game";
import { errorKey } from "@/shared/api/client";
import {
  useGameplayStatus,
  useInvalidateGameplay,
} from "@/features/game/useGameplayStatus";
import {
  announceStageUpdate,
  announceTimeUpdate,
} from "@/features/game/gameEvents";
import type { ReportResult, ReportType } from "@/shared/types/api";

const REPORT_TYPES: ReportType[] = [
  "clue_report",
  "suspect_report",
  "progress_report",
  "incident_report",
  "final_report",
];

/**
 * The Report Center (/missions/:id/report): file typed reports to command.
 * Accepted reports are real game moves — they cost mission time, advance
 * stages, and unlock content; rejections explain exactly what is missing.
 */
export function ReportTerminalPage() {
  const { missionId = "" } = useParams();
  const { t } = useI18n();
  const status = useGameplayStatus(missionId);
  const invalidate = useInvalidateGameplay(missionId);

  const [type, setType] = useState<ReportType>("progress_report");
  const [summary, setSummary] = useState("");
  const [linked, setLinked] = useState<string[]>([]);
  const [suspectId, setSuspectId] = useState("");
  const [result, setResult] = useState<ReportResult | null>(null);

  const history = useQuery({
    queryKey: ["reports", missionId],
    queryFn: () => reportsApi.list(missionId),
    enabled: !!missionId,
  });

  const submit = useMutation({
    mutationFn: () =>
      reportsApi.submit(missionId, {
        type,
        summary,
        linked_clue_ids: linked,
        ...(suspectId ? { suspect_character_id: suspectId } : {}),
      }),
    onSuccess: (res) => {
      setResult(res);
      setSummary("");
      setLinked([]);
      announceTimeUpdate(res.time_update);
      announceStageUpdate(res.stage_update);
      invalidate();
      void history.refetch();
    },
  });

  const clues = status.data?.clues ?? [];
  const characters = (status.data?.characters ?? []).filter(
    (c) => c.category !== "guide",
  );
  const pendingType = status.data?.pending_report_type;

  const toggleClue = (id: string) =>
    setLinked((cur) =>
      cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id],
    );

  return (
    <div className="report-terminal">
      <header className="report-terminal-head">
        <FileText size={20} aria-hidden />
        <div>
          <h2>{t("report.title")}</h2>
          <p className="muted">{t("report.subtitle")}</p>
        </div>
      </header>

      <section className="report-form panel">
        <label className="report-label">{t("report.type")}</label>
        <div className="report-type-row">
          {REPORT_TYPES.map((rt) => (
            <button
              key={rt}
              className={`report-type-chip${type === rt ? " on" : ""}${
                pendingType === rt ? " pending" : ""
              }`}
              onClick={() => setType(rt)}
            >
              {t(`report.type.${rt}`)}
            </button>
          ))}
        </div>

        <label className="report-label" htmlFor="report-summary">
          {t("report.summary")}
        </label>
        <textarea
          id="report-summary"
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
          placeholder={t("report.summaryPlaceholder")}
          rows={4}
          maxLength={4000}
        />

        {(type === "clue_report" || type === "suspect_report") && (
          <>
            <label className="report-label">{t("report.linkEvidence")}</label>
            <p className="muted" style={{ fontSize: 12, margin: "0 0 6px" }}>
              {t("report.linkEvidenceHint")}
            </p>
            <div className="report-evidence-list">
              {clues.map((c) => (
                <button
                  key={c.id}
                  className={`report-evidence${linked.includes(c.id) ? " on" : ""}`}
                  onClick={() => toggleClue(c.id)}
                >
                  <span className={`evidence-status ${c.status ?? "discovered"}`} />
                  {c.title}
                </button>
              ))}
            </div>
          </>
        )}

        {type === "suspect_report" && (
          <>
            <label className="report-label" htmlFor="report-suspect">
              {t("report.suspectSelect")}
            </label>
            <select
              id="report-suspect"
              value={suspectId}
              onChange={(e) => setSuspectId(e.target.value)}
            >
              <option value="">{t("report.suspectNone")}</option>
              {characters.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name} — {c.role}
                </option>
              ))}
            </select>
          </>
        )}

        <div style={{ marginTop: 14 }}>
          <GameButton
            variant="mission"
            disabled={!summary.trim() || submit.isPending}
            onClick={() => submit.mutate()}
          >
            {t("report.submit")}
          </GameButton>
        </div>
        {submit.isError && (
          <p className="report-error">{t(errorKey(submit.error))}</p>
        )}
      </section>

      {result && (
        <section
          className={`report-verdict panel ${result.verdict}`}
          aria-live="assertive"
        >
          <div className="report-verdict-head">
            {result.verdict === "accepted" ? (
              <CheckCircle2 size={20} aria-hidden />
            ) : (
              <XCircle size={20} aria-hidden />
            )}
            <strong>
              {result.verdict === "accepted"
                ? t("report.accepted")
                : t("report.rejected")}
            </strong>
          </div>
          <p>{result.feedback}</p>
          {result.missing_requirements.length > 0 && (
            <>
              <div className="report-label">{t("report.missing")}</div>
              <ul className="report-missing">
                {result.missing_requirements.map((m) => (
                  <li key={m}>{m}</li>
                ))}
              </ul>
            </>
          )}
        </section>
      )}

      <section className="report-history panel">
        <h3>{t("report.history")}</h3>
        {(history.data ?? []).length === 0 && (
          <p className="muted">{t("report.none")}</p>
        )}
        {(history.data ?? []).map((r) => (
          <div key={r.id} className={`report-history-row ${r.verdict}`}>
            <span className="report-history-type">{t(`report.type.${r.type}`)}</span>
            <span className="report-history-verdict">
              {r.verdict === "accepted" ? t("report.accepted") : t("report.rejected")}
            </span>
            <span className="report-history-summary">{r.summary}</span>
          </div>
        ))}
      </section>
    </div>
  );
}
