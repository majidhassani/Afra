import { useState, type FormEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Radio, SendHorizonal } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useLanguageGuard } from "@/shared/i18n/languageGuard";
import { guidanceApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { CostBadge } from "@/shared/ui/badges";
import { Analyzing } from "@/shared/ui/game";
import { Button } from "@/shared/ui/Button";
import type { GuidanceContext, GuidanceResult } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

interface Props {
  missionId: string;
  /** Where in the app this composer lives (sent to the backend as context). */
  screen: string;
  locationId?: string;
  selectedClueId?: string;
}

const SUGGESTIONS: TranslationKey[] = [
  "guidance.suggest.next",
  "guidance.suggest.clue",
  "guidance.suggest.time",
  "guidance.suggest.summary",
];

/** "Ask Mission Control" — the always-available AI assistant. */
export function GuidancePanel({
  missionId,
  screen,
  locationId,
  selectedClueId,
}: Props) {
  const { t } = useI18n();
  const guardLanguage = useLanguageGuard();
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
      guardLanguage(res.message);
      setResult(res);
      setMessage("");
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  const ask = (msg: string) => {
    const trimmed = msg.trim();
    if (trimmed && !mutation.isPending) mutation.mutate(trimmed);
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    ask(message);
  };

  return (
    <section aria-label={t("guidance.title")}>
      <div className="row" style={{ marginBottom: 10, gap: 8 }}>
        <span
          className="avatar avatar-ph2"
          style={{
            width: 30,
            height: 30,
            background: "var(--accent-ai-dim)",
            color: "var(--accent-ai)",
            border: "1px solid rgba(125,211,199,0.35)",
          }}
          aria-hidden
        >
          <Radio size={15} />
        </span>
        <h3>{t("guidance.title")}</h3>
      </div>

      {/* Suggested questions */}
      <div className="suggest-chips">
        {SUGGESTIONS.map((key) => (
          <button
            key={key}
            type="button"
            className="suggest-chip"
            disabled={mutation.isPending}
            onClick={() => ask(t(key))}
          >
            {t(key)}
          </button>
        ))}
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
          <Button
            variant="ghost"
            size="sm"
            type="submit"
            loading={mutation.isPending}
            disabled={!message.trim()}
          >
            <SendHorizonal size={14} className="rtl-flip" aria-hidden />
            {t("guidance.ask")}
          </Button>
        </div>
      </form>

      {mutation.isPending && <Analyzing label={t("guidance.analyzing")} />}
      {mutation.isError && (
        <p className="field-error" role="alert" style={{ marginTop: 10 }}>
          {t(errorKey(mutation.error))}
        </p>
      )}
      {result && !mutation.isPending && (
        <div className="mc-briefing" style={{ marginTop: 12 }}>
          <div className="mc-head">
            <span className="mc-live" aria-hidden />
            {t("guidance.title")}
            <span className="mc-hint">{result.hint_level}</span>
          </div>
          <div className="mc-body" style={{ unicodeBidi: "plaintext" }}>
            {result.message}
          </div>
          <div className="mc-foot">
            {result.referenced_items.map((item) => (
              <span key={`${item.type}-${item.id}`} className="mc-ref">
                {item.name}
              </span>
            ))}
            <span className="grow" />
            <CostBadge coins={result.cost.coins_charged} />
          </div>
        </div>
      )}
    </section>
  );
}
