import type { MouseEventHandler, ReactNode } from "react";
import { Link } from "react-router-dom";

/**
 * Button — the canonical button primitive for the whole app.
 *
 * `GameButton` (shared/ui/game.tsx) and `TacticalButton` (shared/ui/avds.tsx)
 * used to be two separate component implementations that happened to render
 * visually-similar buttons. Both are now thin wrappers around this single
 * component: they translate their own (unchanged) `variant` prop into one
 * of the variants below and delegate rendering here. No call site of either
 * wrapper needs to change.
 *
 * Variants:
 *  - "default":  restrained neutral action. Not currently used by the two
 *                legacy wrappers (they keep their own historical primary
 *                treatment below) — this is the variant new call sites
 *                should reach for by default.
 *  - "ghost":    low-emphasis secondary action.
 *  - "danger":   destructive / high-risk action (semantic red).
 *  - "mission":  tactical green glow, angular cut corner — a confirmed
 *                gameplay action (accept mission, confirm objective).
 *  - "tactical": restrained teal/cyan gameplay CTA (inspect, investigate,
 *                navigate) — GameButton/TacticalButton's "primary".
 *  - "reward":   gold accent — claiming rewards, wallet-adjacent actions.
 */
export type ButtonVariant = "default" | "ghost" | "danger" | "mission" | "tactical" | "reward";

const variantClass: Record<ButtonVariant, string> = {
  default: "ui-btn-default",
  ghost: "ui-btn-ghost",
  danger: "ui-btn-danger",
  mission: "ui-btn-mission",
  tactical: "ui-btn-tactical",
  reward: "ui-btn-reward",
};

export function Button({
  children,
  variant = "default",
  to,
  size,
  type = "button",
  disabled,
  onClick,
  title,
  className = "",
  legacyClassName = "",
}: {
  children: ReactNode;
  variant?: ButtonVariant;
  /** Renders as a router Link instead of a <button> when provided. */
  to?: string;
  size?: "sm";
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: MouseEventHandler<HTMLButtonElement | HTMLAnchorElement>;
  title?: string;
  className?: string;
  /**
   * @internal Extra class name(s) GameButton/TacticalButton pass through
   * for backward compatibility (their historical `.game-btn*` /
   * `.av-tactical-button*` classes). New call sites should not use this —
   * every property those classes set is already covered by `variant`.
   */
  legacyClassName?: string;
}) {
  const cls = ["ui-btn", variantClass[variant], size ? "sm" : "", legacyClassName, className]
    .filter(Boolean)
    .join(" ");

  if (to) {
    return (
      <Link className={cls} to={to} title={title}>
        {children}
      </Link>
    );
  }

  return (
    <button type={type} className={cls} disabled={disabled} onClick={onClick} title={title}>
      {children}
    </button>
  );
}
