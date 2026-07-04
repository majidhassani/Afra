import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Sparkles, SendHorizonal } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { guidanceApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { CostBadge } from "@/shared/ui/badges";
import type { GuidanceContext, GuidanceResult } from "@/shared/types/api";

interface Props {
  missionId: string;
  /** Where in the app this composer lives (sent to the backend as context). */
  screen: string;
  locationId?: string;
  selectedClueId?: string;
}

/** Reusable "Ask Mission Control" composer + response panel. */
export function GuidancePanel({
  missionId,
  screen,
  locationId,
  selectedClueId,
}: Props) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [message, setMessage] = useState("");
  const [result, setResult] = useState<GuidanceResult | null>(null);

  const mutation = useMutation({
    mutationFn: (msg: string) => {
      const context: GuidanceContext = {
        screen,
        ...(locationId ? { location_id: locationId } : {}),
        ...(selectedClueId ? { selected_clue_id: selectedClueId } : {}),
      };
      return guidanceApi.ask(missionId, msg, context);
    },
    onSuccess: (res) => {
      setResult(res);
      setMessage("");
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = message.trim();
    if (trimmed) mutation.mutate(trimmed);
  };

  return (
    <section aria-label={t("guidance.title")}>
      <div className="row" style={{ marginBottom: 10 }}>
        <Sparkles size={15} color="var(--accent-ai)" aria-hidden />
        <h3>{t("guidance.title")}</h3>
      </div>
      <form onSubmit={onSubmit} className="stack" style={{ gap: 8 }}>
        <textarea
          className="textarea"
          rows={2}
          maxLength={2000}
          placeholder={t("guidance.placeholder")}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          aria-label={t("guidance.placeholder")}
        />
        <div className="spread">
          <span className="faint">{t("guidance.disclaimer")}</span>
          <button
            className="btn btn-secondary"
            type="submit"
            disabled={mutation.isPending || !message.trim()}
          >
            <SendHorizonal size={14} aria-hidden />
            {t("guidance.ask")}
          </button>
        </div>
      </form>
      {mutation.isPending && (
        <div className="skeleton" style={{ height: 56, marginTop: 10 }} />
      )}
      {mutation.isError && (
        <p className="field-error" role="alert" style={{ marginTop: 10 }}>
          {t(errorKey(mutation.error))}
        </p>
      )}
      {result && !mutation.isPending && (
        <div className="stack" style={{ marginTop: 10, gap: 8 }}>
          <div className="guidance-answer">{result.message}</div>
          <div className="row" style={{ flexWrap: "wrap" }}>
            <span className="chip chip-ai">
              {t("guidance.hintLevel")}: {result.hint_level}
            </span>
            <CostBadge coins={result.cost.coins_charged} />
            {result.referenced_items.map((item) => (
              <span key={`${item.type}-${item.id}`} className="chip">
                {item.name}
              </span>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
