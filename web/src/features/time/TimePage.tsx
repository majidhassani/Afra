import { useState } from "react";
import { useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Clock3, FastForward, AlertTriangle } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { timeApi, walletApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows } from "@/shared/ui/states";
import { CostBadge } from "@/shared/ui/badges";
import { GameButton } from "@/shared/ui/game";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import type { TimeAdvanceResult, TimeUnit } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

const UNITS: TimeUnit[] = ["minutes", "hours", "days"];

export function TimePage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [amount, setAmount] = useState(1);
  const [unit, setUnit] = useState<TimeUnit>("hours");
  const [result, setResult] = useState<TimeAdvanceResult | null>(null);

  const time = useQuery({
    queryKey: ["mission", missionId, "time"],
    queryFn: () => timeApi.get(missionId!),
    enabled: !!missionId,
  });

  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
  });

  const advance = useMutation({
    mutationFn: () => timeApi.advance(missionId!, amount, unit),
    onSuccess: (res) => {
      setResult(res);
      void queryClient.invalidateQueries({ queryKey: ["mission", missionId] });
      void queryClient.invalidateQueries({
        queryKey: ["mission", missionId, "time"],
      });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  const publicState =
    time.data?.public_state && typeof time.data.public_state === "object"
      ? Object.entries(time.data.public_state)
      : [];

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("time.title")}</h1>
      </header>

      {time.isPending && <SkeletonRows rows={5} />}
      {time.isError && (
        <ErrorState error={time.error} onRetry={() => time.refetch()} />
      )}

      {time.isSuccess && (
        <div className="dash-grid">
          <section className="panel col-8">
            <div className="band">
              <div className="band-title">{t("time.current")}</div>
              <div className="row">
                <Clock3 size={22} color="var(--accent-ai)" aria-hidden />
                <span style={{ fontSize: 24, fontWeight: 600 }} className="mono-num">
                  {time.data.current_time}
                </span>
              </div>
            </div>

            <div className="band" style={{ borderBottom: "none" }}>
              <div className="band-title">{t("time.advance")}</div>
              <p className="row muted" style={{ marginBottom: 12 }}>
                <AlertTriangle size={14} color="var(--accent-wallet)" aria-hidden />
                {t("time.warning")}
              </p>
              <div className="row" style={{ flexWrap: "wrap", gap: 12 }}>
                <div className="field" style={{ width: 110 }}>
                  <label className="field-label" htmlFor="amount">
                    {t("time.amount")}
                  </label>
                  <input
                    id="amount"
                    className="input mono-num"
                    type="number"
                    min={1}
                    max={99}
                    value={amount}
                    onChange={(e) =>
                      setAmount(Math.max(1, Number(e.target.value) || 1))
                    }
                  />
                </div>
                <div className="field">
                  <span className="field-label">{t("time.unit")}</span>
                  <div className="segmented">
                    {UNITS.map((u) => (
                      <button
                        key={u}
                        type="button"
                        aria-pressed={unit === u}
                        onClick={() => setUnit(u)}
                      >
                        {t(`time.unit.${u}` as TranslationKey)}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="field">
                  <span className="field-label">&nbsp;</span>
                  <div className="row">
                    <GameButton
                      variant="mission"
                      disabled={advance.isPending}
                      onClick={() => advance.mutate()}
                    >
                      <FastForward size={14} aria-hidden />
                      {t("time.advance")}
                    </GameButton>
                    {pricing.data?.advance_time !== undefined && (
                      <CostBadge coins={pricing.data.advance_time} />
                    )}
                  </div>
                </div>
              </div>

              {advance.isPending && (
                <div className="skeleton" style={{ height: 56, marginTop: 14 }} />
              )}
              {advance.isError && (
                <p className="field-error" role="alert" style={{ marginTop: 14 }}>
                  {t(errorKey(advance.error))}
                </p>
              )}
              {result && !advance.isPending && (
                <div className="stack" style={{ marginTop: 14 }}>
                  <div className="band-title">{t("time.summary")}</div>
                  <p style={{ unicodeBidi: "plaintext" }}>{result.summary}</p>
                  {result.events.length > 0 && (
                    <>
                      <div className="band-title">{t("time.events")}</div>
                      <div className="row" style={{ flexWrap: "wrap" }}>
                        {result.events.map((ev, i) => (
                          <span key={`${ev.type}-${i}`} className="chip chip-mission">
                            {ev.title}
                          </span>
                        ))}
                      </div>
                    </>
                  )}
                  <CostBadge coins={result.cost.coins_charged} />
                </div>
              )}
            </div>
          </section>

          <div className="col-4 stack">
            {publicState.length > 0 && (
              <section className="panel" aria-label={t("time.situation")}>
                <div className="band-title" style={{ padding: "14px 16px 0" }}>
                  {t("time.situation")}
                </div>
                <div className="item-list">
                  {publicState.map(([key, value]) => (
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
                <GuidancePanel missionId={missionId!} screen="time" />
              </div>
            </section>
          </div>
        </div>
      )}
    </div>
  );
}
