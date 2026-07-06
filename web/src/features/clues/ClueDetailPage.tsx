import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Microscope, Lightbulb, ScanSearch } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useLanguageGuard } from "@/shared/i18n/languageGuard";
import { cluesApi, walletApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows } from "@/shared/ui/states";
import { CostBadge, Meter } from "@/shared/ui/badges";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import { importanceChip, importanceLabel } from "./CluesPage";
import type {
  ClueExplainResult,
  ClueInspectResult,
} from "@/shared/types/api";

export function ClueDetailPage() {
  const { missionId, clueId } = useParams<{ missionId: string; clueId: string }>();
  const { t } = useI18n();
  const guardLanguage = useLanguageGuard();
  const queryClient = useQueryClient();
  const [question, setQuestion] = useState("");
  const [inspectResult, setInspectResult] = useState<ClueInspectResult | null>(null);
  const [explainResult, setExplainResult] = useState<ClueExplainResult | null>(null);

  const clue = useQuery({
    queryKey: ["mission", missionId, "clue", clueId],
    queryFn: () => cluesApi.detail(missionId!, clueId!),
    enabled: !!missionId && !!clueId,
  });

  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
  });

  const inspect = useMutation({
    mutationFn: () =>
      cluesApi.inspect(missionId!, clueId!, question.trim() || undefined),
    onSuccess: (result) => {
      guardLanguage(result.analysis);
      setInspectResult(result);
      setQuestion("");
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "clue", clueId],
      });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  const explain = useMutation({
    mutationFn: () => cluesApi.explain(missionId!, clueId!),
    onSuccess: (result) => {
      guardLanguage(result.explanation);
      setExplainResult(result);
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  if (clue.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={7} />
      </div>
    );
  }
  if (clue.isError) {
    return (
      <div className="page">
        <ErrorState error={clue.error} onRetry={() => clue.refetch()} />
      </div>
    );
  }

  const data = clue.data;
  const publicData =
    data.public_data && typeof data.public_data === "object"
      ? Object.entries(data.public_data).filter(([key]) => {
          const safeKey = key.toLowerCase();
          return (
            !safeKey.includes("prompt") &&
            !safeKey.includes("private") &&
            !safeKey.includes("hidden") &&
            !safeKey.includes("truth")
          );
        })
      : [];

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <Link
            className="row faint"
            to={`/app/missions/${missionId}/clues`}
            style={{ marginBottom: 6 }}
          >
            <ArrowLeft size={13} className="rtl-flip" aria-hidden />
            {t("clues.title")}
          </Link>
          <h1>{data.title}</h1>
          <div className="row subtitle" style={{ flexWrap: "wrap" }}>
            <span className="chip">{data.type}</span>
            <span className={`chip ${importanceChip(data.importance)}`}>
              {t("clues.importance")}: {importanceLabel(t, data.importance)}
            </span>
            <span className="row faint" style={{ gap: 6 }}>
              {t("clues.reliability")}
              <Meter value={data.reliability} color="var(--accent-wallet)" />
              <span className="mono-num">{data.reliability}%</span>
            </span>
          </div>
        </div>
      </header>

      <div className="dash-grid">
        <section className="panel col-8">
          <div className="band">
            <div className="evidence-figure" aria-hidden>
              <ScanSearch size={30} />
              <span>{data.type.replace(/_/g, " ")}</span>
            </div>
            <p style={{ unicodeBidi: "plaintext" }}>{data.detailed_description}</p>
          </div>

          <div className="band">
            <div className="band-title">{t("clues.inspect")}</div>
            <div className="stack" style={{ gap: 8 }}>
              <label className="field-label" htmlFor="inspect-q">
                {t("clues.inspect.question")}
              </label>
              <textarea
                id="inspect-q"
                className="textarea"
                rows={2}
                value={question}
                placeholder={t("clues.inspect.placeholder")}
                onChange={(e) => setQuestion(e.target.value)}
              />
              <div className="row">
                <button
                  className="btn btn-secondary"
                  disabled={inspect.isPending}
                  onClick={() => inspect.mutate()}
                >
                  <Microscope size={14} aria-hidden />
                  {t("clues.inspect")}
                </button>
                {pricing.data?.clue_inspect !== undefined && (
                  <CostBadge coins={pricing.data.clue_inspect} />
                )}
              </div>
            </div>
            {inspect.isPending && (
              <div className="skeleton" style={{ height: 48, marginTop: 12 }} />
            )}
            {inspect.isError && (
              <p className="field-error" role="alert" style={{ marginTop: 12 }}>
                {t(errorKey(inspect.error))}
              </p>
            )}
            {inspectResult && !inspect.isPending && (
              <div className="stack" style={{ marginTop: 12 }}>
                <div className="band-title">{t("clues.analysis")}</div>
                <p style={{ unicodeBidi: "plaintext" }}>{inspectResult.analysis}</p>
                {inspectResult.new_facts.length > 0 && (
                  <ul style={{ margin: 0, paddingInlineStart: 18 }}>
                    {inspectResult.new_facts.map((fact) => (
                      <li key={fact} className="muted">
                        {fact}
                      </li>
                    ))}
                  </ul>
                )}
                <CostBadge coins={inspectResult.cost.coins_charged} />
              </div>
            )}
          </div>

          <div className="band" style={{ borderBottom: "none" }}>
            <div className="band-title">{t("clues.explain")}</div>
            <div className="row">
              <button
                className="btn btn-secondary"
                disabled={explain.isPending}
                onClick={() => explain.mutate()}
              >
                <Lightbulb size={14} aria-hidden />
                {t("clues.explain")}
              </button>
              {pricing.data?.clue_explain !== undefined && (
                <CostBadge coins={pricing.data.clue_explain} />
              )}
            </div>
            {explain.isPending && (
              <div className="skeleton" style={{ height: 48, marginTop: 12 }} />
            )}
            {explain.isError && (
              <p className="field-error" role="alert" style={{ marginTop: 12 }}>
                {t(errorKey(explain.error))}
              </p>
            )}
            {explainResult && !explain.isPending && (
              <div className="stack" style={{ marginTop: 12 }}>
                <p style={{ unicodeBidi: "plaintext" }}>{explainResult.explanation}</p>
                {explainResult.next_steps.length > 0 && (
                  <>
                    <div className="band-title">{t("clues.nextSteps")}</div>
                    <ul style={{ margin: 0, paddingInlineStart: 18 }}>
                      {explainResult.next_steps.map((step) => (
                        <li key={step} className="muted">
                          {step}
                        </li>
                      ))}
                    </ul>
                  </>
                )}
                {explainResult.compare_with.length > 0 && (
                  <>
                    <div className="band-title">{t("clues.compareWith")}</div>
                    <div className="row" style={{ flexWrap: "wrap" }}>
                      {explainResult.compare_with.map((item) => (
                        <span key={item} className="chip">
                          {item}
                        </span>
                      ))}
                    </div>
                  </>
                )}
                <CostBadge coins={explainResult.cost.coins_charged} />
              </div>
            )}
          </div>
        </section>

        <div className="col-4 stack">
          {publicData.length > 0 && (
            <section className="panel" aria-label={t("clues.publicData")}>
              <div className="band-title" style={{ padding: "14px 16px 0" }}>
                {t("clues.publicData")}
              </div>
              <div className="item-list">
                {publicData.map(([key, value]) => (
                  <div key={key} className="item-row">
                    <span className="grow sub">{key}</span>
                    <span style={{ fontSize: 13, unicodeBidi: "plaintext" }}>
                      {String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}
          <section className="panel">
            <div style={{ padding: 16 }}>
              <GuidancePanel
                missionId={missionId!}
                screen="clue_detail"
                selectedClueId={clueId}
              />
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
