import type { CSSProperties, MouseEventHandler, ReactNode } from "react";
import { Link } from "react-router-dom";
import { Loader2 } from "lucide-react";

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
 *  - "default":   restrained neutral action — solid high-contrast fill,
 *                 mirrors base.css's .btn-primary exactly (most ordinary
 *                 forms: auth, settings, journal).
 *  - "secondary": elevated neutral action, mirrors base.css's .btn-secondary.
 *  - "subtle":    lowest-emphasis text-only action, mirrors base.css's
 *                 .btn-ghost (e.g. a cancel/dismiss next to a primary action).
 *  - "ghost":     ⚠ NOT the same as "subtle" — this is GameButton/
 *                 TacticalButton's own historical ghost treatment (a filled
 *                 low-contrast chip, not text-only). Kept distinct so
 *                 migrating those two wrappers to this component didn't
 *                 change their rendered output. New call sites outside
 *                 gameplay screens should use "subtle" instead.
 *  - "danger":    destructive / high-risk action (semantic red) — used by
 *                 both raw call sites and TacticalButton's danger variant.
 *  - "mission":   tactical green glow, angular cut corner — a confirmed
 *                 gameplay action (accept mission, confirm objective).
 *  - "tactical":  restrained teal/cyan gameplay CTA (inspect, investigate,
 *                 navigate) — GameButton/TacticalButton's "primary".
 *  - "reward":    gold accent — claiming rewards, wallet-adjacent actions.
 */
export type ButtonVariant =
  | "default"
  | "secondary"
  | "subtle"
  | "ghost"
  | "danger"
  | "mission"
  | "tactical"
  | "reward";

const variantClass: Record<ButtonVariant, string> = {
  default: "ui-btn-default",
  secondary: "ui-btn-secondary",
  subtle: "ui-btn-subtle",
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
  loading = false,
  fullWidth = false,
  onClick,
  title,
  ariaLabel,
  className = "",
  style,
  legacyClassName = "",
}: {
  children: ReactNode;
  variant?: ButtonVariant;
  /** Renders as a router Link instead of a <button> when provided. */
  to?: string;
  size?: "sm";
  type?: "button" | "submit";
  disabled?: boolean;
  /**
   * Shows an inline spinner and marks the button busy/disabled — for async
   * actions (form submits, mutations) where the UI needs to confirm
   * "something is happening" rather than sitting silent after a click.
   */
  loading?: boolean;
  /** Stretches the button to fill its container (common for form submits). */
  fullWidth?: boolean;
  onClick?: MouseEventHandler<HTMLButtonElement | HTMLAnchorElement>;
  title?: string;
  /** For icon-only buttons that need an accessible name. */
  ariaLabel?: string;
  className?: string;
  /** One-off inline layout tweaks (spacing, etc.) at individual call sites. */
  style?: CSSProperties;
  /**
   * @internal Extra class name(s) GameButton/TacticalButton pass through
   * for backward compatibility (their historical `.game-btn*` /
   * `.av-tactical-button*` classes). New call sites should not use this —
   * every property those classes set is already covered by `variant`.
   */
  legacyClassName?: string;
}) {
  const cls = [
    "ui-btn",
    variantClass[variant],
    size ? "sm" : "",
    fullWidth ? "ui-btn-full" : "",
    loading ? "ui-btn-loading" : "",
    legacyClassName,
    className,
  ]
    .filter(Boolean)
    .join(" ");
  const isDisabled = disabled || loading;

  // Only wrap children when loading — in the (overwhelmingly common)
  // non-loading case this renders `children` completely untouched, so
  // every existing GameButton/TacticalButton call site's flex layout
  // (icon + text spaced by the button's own `gap`) is unaffected.
  const content = loading ? (
    <>
      <Loader2 className="ui-btn-spinner spin" size={15} aria-hidden />
      {children}
    </>
  ) : (
    children
  );

  if (to) {
    return (
      <Link
        className={cls}
        style={style}
        to={to}
        title={title}
        aria-label={ariaLabel}
        aria-disabled={isDisabled || undefined}
        tabIndex={isDisabled ? -1 : undefined}
        onClick={isDisabled ? (e) => e.preventDefault() : onClick}
      >
        {content}
      </Link>
    );
  }

  return (
    <button
      type={type}
      className={cls}
      style={style}
      disabled={isDisabled}
      onClick={onClick}
      title={title}
      aria-label={ariaLabel}
      aria-busy={loading || undefined}
    >
      {content}
    </button>
  );
}
