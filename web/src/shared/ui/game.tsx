import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";
import { Button, type ButtonVariant } from "./Button";

const gameButtonVariant: Record<"primary" | "mission" | "ghost", ButtonVariant> = {
  primary: "tactical",
  mission: "mission",
  ghost: "ghost",
};

/**
 * Primary game action button with tactical glow. Thin wrapper around the
 * canonical Button primitive (shared/ui/Button.tsx) — kept as its own
 * component so every existing call site's props/className stay unchanged.
 */
export function GameButton({
  children,
  variant = "primary",
  size,
  type = "button",
  disabled,
  onClick,
  title,
  to,
  ariaLabel,
}: {
  children: ReactNode;
  variant?: "primary" | "mission" | "ghost";
  size?: "sm";
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: () => void;
  title?: string;
  /** Renders as a link (via Button's own `to` support) instead of a <button>.
   * Use this rather than wrapping <GameButton> in an outer <Link> — nesting
   * a <button> inside an <a> is invalid HTML and breaks keyboard/AT focus. */
  to?: string;
  ariaLabel?: string;
}) {
  return (
    <Button
      variant={gameButtonVariant[variant]}
      size={size}
      type={type}
      disabled={disabled}
      onClick={onClick}
      title={title}
      to={to}
      ariaLabel={ariaLabel}
      legacyClassName={`game-btn game-btn-${variant}`}
    >
      {children}
    </Button>
  );
}

/** Color-coded 0-100 risk meter. */
export function RiskMeter({ value }: { value: number }) {
  const v = Math.max(0, Math.min(100, value));
  const band = v >= 66 ? "risk-high" : v >= 33 ? "risk-med" : "risk-low";
  return (
    <span className={`risk-meter ${band}`} role="img" aria-label={`risk ${v}%`}>
      <span style={{ width: `${v}%` }} />
    </span>
  );
}

/** 0-100 objective/mission progress bar. */
export function ObjectiveProgress({ value }: { value: number }) {
  const v = Math.max(0, Math.min(100, value));
  return (
    <span className="obj-progress" role="img" aria-label={`${v}%`}>
      <span style={{ width: `${v}%` }} />
    </span>
  );
}

/** One HUD stat tile. */
export function HudStat({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="hud-stat">
      <span className="hud-label">{label}</span>
      <div className="hud-value">{children}</div>
    </div>
  );
}

/** Cinematic full-panel loading screen with optional rotating steps. */
export function LoadingScreen({
  title,
  steps,
  activeStep,
}: {
  title: string;
  steps?: string[];
  activeStep?: number;
}) {
  return (
    <div className="loading-screen">
      <div className="loading-orb" aria-hidden />
      <div className="loading-title">{title}</div>
      {steps && steps.length > 0 && (
        <div className="loading-steps">
          {steps.map((s, i) => (
            <span key={s} className={`step${activeStep === i ? " on" : ""}`}>
              {s}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

/** Inline "analyzing" indicator for AI responses. */
export function Analyzing({ label }: { label: string }) {
  return (
    <div className="row" style={{ gap: 8, color: "var(--accent-ai)", padding: "8px 0" }}>
      <Loader2 size={15} className="spin" aria-hidden />
      <span style={{ fontSize: 13 }}>{label}</span>
    </div>
  );
}

/** Game-economy coin balance chip. */
export function WalletBalance({ balance }: { balance: number | undefined }) {
  return (
    <span className="coin-balance">
      <span className="coin-dot" aria-hidden />
      {balance ?? "—"}
    </span>
  );
}

/**
 * Canonical color for a TimelineItem's `importance` field.
 *
 * Found during the full-repo audit: three different places derived a color
 * from the same "high" | "medium" | "low" importance value, and each picked
 * a different mapping — TimelineLog.tsx used cyan/green, ScrollableTimeline's
 * CSS used red/cyan, and MissionDashboardPage's inline TimelineRail mapping
 * used red/gold/green. Same data field, three contradictory meanings
 * depending which timeline you were looking at. This is the single source of
 * truth now: gold = high priority (matches the "high-priority evidence"
 * meaning gold already carries elsewhere), cyan = medium/notable, green =
 * low/routine. Red is intentionally not used here — it stays reserved for
 * genuinely dangerous/failed events, which already get their own icon
 * (CircleX) rather than borrowing the importance color.
 */
export function importanceTone(
  importance: "high" | "medium" | "low" | string,
): "gold" | "cyan" | "green" {
  if (importance === "high") return "gold";
  if (importance === "medium") return "cyan";
  return "green";
}
