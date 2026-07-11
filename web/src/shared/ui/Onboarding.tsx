import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  Radio,
  Rocket,
  Map,
  Users,
  Search,
  Sparkles,
  Coins,
  X,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import type { TranslationKey } from "@/shared/i18n/en";

const SEEN_KEY = "afra.onboarding.seen";
const REPLAY_EVENT = "afra:onboarding:replay";

/**
 * Clears the "seen" flag and asks any mounted <Onboarding> to reopen.
 * Client/session-only — there is no persisted server-side flag for this
 * yet, so the walkthrough naturally replays every fresh session until
 * that's wired up. Settings' "Replay tutorial" action calls this.
 */
export function replayOnboarding() {
  try {
    sessionStorage.removeItem(SEEN_KEY);
  } catch {
    // Storage may be unavailable (private mode, disabled) — the event
    // below still reopens the walkthrough for this tab either way.
  }
  window.dispatchEvent(new Event(REPLAY_EVENT));
}

interface Step {
  key: string;
  icon: typeof Rocket;
  titleKey: TranslationKey;
  bodyKey: TranslationKey;
}

const STEPS: Step[] = [
  { key: "welcome", icon: Radio, titleKey: "onboarding.welcome.title", bodyKey: "onboarding.welcome.body" },
  { key: "missions", icon: Rocket, titleKey: "onboarding.missions.title", bodyKey: "onboarding.missions.body" },
  { key: "map", icon: Map, titleKey: "onboarding.map.title", bodyKey: "onboarding.map.body" },
  { key: "characters", icon: Users, titleKey: "onboarding.characters.title", bodyKey: "onboarding.characters.body" },
  { key: "clues", icon: Search, titleKey: "onboarding.clues.title", bodyKey: "onboarding.clues.body" },
  { key: "timeline", icon: Radio, titleKey: "onboarding.timeline.title", bodyKey: "onboarding.timeline.body" },
  { key: "guidance", icon: Sparkles, titleKey: "onboarding.guidance.title", bodyKey: "onboarding.guidance.body" },
  { key: "rewards", icon: Coins, titleKey: "onboarding.rewards.title", bodyKey: "onboarding.rewards.body" },
];

/**
 * First-time agent activation walkthrough. Mounted once in AppShell and
 * gated to routes outside an active mission (see AppShell) so it never
 * interrupts real gameplay. Entirely client/session-state — see
 * `replayOnboarding` above for why there's no server persistence yet.
 */
export function Onboarding({ eligible }: { eligible: boolean }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState(0);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!eligible) return;
    let seen: boolean;
    try {
      seen = sessionStorage.getItem(SEEN_KEY) === "1";
    } catch {
      seen = false;
    }
    if (!seen) setOpen(true);
  }, [eligible]);

  useEffect(() => {
    const onReplay = () => {
      setStep(0);
      setOpen(true);
    };
    window.addEventListener(REPLAY_EVENT, onReplay);
    return () => window.removeEventListener(REPLAY_EVENT, onReplay);
  }, []);

  useEffect(() => {
    if (open) dialogRef.current?.focus();
  }, [open, step]);

  const finish = () => {
    try {
      sessionStorage.setItem(SEEN_KEY, "1");
    } catch {
      // Best-effort only — worst case the walkthrough replays next visit.
    }
    setOpen(false);
  };

  const current = STEPS[step];
  const isLast = step === STEPS.length - 1;
  const stepLabel = useMemo(
    () => t("onboarding.stepOf", { current: step + 1, total: STEPS.length }),
    [step, t],
  );

  if (!open) return null;

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      finish();
      return;
    }
    if (e.key === "Tab") {
      // Simple focus trap: the dialog's own focusable set is small and
      // static (skip, back?, next/done), so wrapping first<->last covers it
      // without pulling in a dedicated focus-trap dependency.
      const root = dialogRef.current;
      if (!root) return;
      const focusables = root.querySelectorAll<HTMLElement>(
        'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
      );
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  };

  // Portaled to <body>: the overlay must be a true viewport layer that no
  // shell-scoped selector, transform or overflow can capture or clip.
  return createPortal(
    <div className="onboarding-scrim" role="presentation">
      <div
        ref={dialogRef}
        className="onboarding-card glass-3 glass-3--ai"
        role="dialog"
        aria-modal="true"
        aria-labelledby="onboarding-title"
        tabIndex={-1}
        onKeyDown={onKeyDown}
      >
        <button
          type="button"
          className="onboarding-close"
          aria-label={t("onboarding.skip")}
          onClick={finish}
        >
          <X size={16} aria-hidden />
        </button>

        <span className="onboarding-icon" aria-hidden>
          <current.icon size={22} />
        </span>
        <h2 id="onboarding-title">{t(current.titleKey)}</h2>
        <p>{t(current.bodyKey)}</p>

        <div className="onboarding-dots" role="img" aria-label={stepLabel}>
          {STEPS.map((s, i) => (
            <span key={s.key} className={i === step ? "on" : ""} aria-hidden />
          ))}
        </div>

        <div className="onboarding-actions">
          <button type="button" className="onboarding-skip" onClick={finish}>
            {t("onboarding.skip")}
          </button>
          <div className="row" style={{ gap: 8 }}>
            {step > 0 && (
              <button
                type="button"
                className="onboarding-back"
                onClick={() => setStep((s) => Math.max(0, s - 1))}
              >
                {t("common.back")}
              </button>
            )}
            <button
              type="button"
              className="onboarding-next"
              onClick={() => (isLast ? finish() : setStep((s) => s + 1))}
            >
              {isLast ? t("onboarding.done") : t("onboarding.next")}
            </button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  );
}
