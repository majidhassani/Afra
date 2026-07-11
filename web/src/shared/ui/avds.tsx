import { useEffect, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode } from "react";
import { Link, NavLink, useLocation } from "react-router-dom";
import {
  Activity,
  AlertTriangle,
  ChevronRight,
  Circle,
  Coins,
  Crosshair,
  Loader2,
  Map,
  MoreHorizontal,
  Radio,
  type LucideIcon,
  Zap,
} from "lucide-react";
import { Button, type ButtonVariant } from "./Button";

export interface GameNavItem {
  to: string;
  icon: LucideIcon;
  label: string;
  end?: boolean;
  badge?: string | number;
}

export interface GameNavGroup { label: string; items: GameNavItem[]; }

export function GameTopBar({
  brand,
  status = "ONLINE",
  userLabel,
  resources,
  actions,
}: {
  brand: string;
  status?: string;
  userLabel?: string;
  resources?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <header className="game-topbar">
      <Link to="/app/dashboard" className="game-brand" aria-label={brand}>
        <span className="game-brand-mark" aria-hidden>
          A
        </span>
        <span className="game-brand-copy">
          <span className="game-brand-name">{brand}</span>
          <span className="game-brand-status">{status}</span>
        </span>
      </Link>
      <div className="game-compass" aria-hidden>
        <span>NW</span>
        <span>N</span>
        <span>NE</span>
        <span>E</span>
        <span>SE</span>
      </div>
      <div className="game-topbar-right">
        <div className="game-resource-strip">{resources}</div>
        {userLabel && <span className="game-agent-label">{userLabel}</span>}
        <div className="game-top-actions">{actions}</div>
      </div>
    </header>
  );
}

export function GameBottomNav({ primaryItems, secondaryGroups, desktopGroups, moreLabel, closeLabel, mode }: { primaryItems: GameNavItem[]; secondaryGroups: GameNavGroup[]; desktopGroups: GameNavGroup[]; moreLabel: string; closeLabel: string; mode: "exploration" | "mission"; }) {
  const [sheet, setSheet] = useState<"closed" | "collapsed" | "expanded">("closed");
  const [dragOffset, setDragOffset] = useState(0);
  const dragStart = useRef(0);
  const moreButton = useRef<HTMLButtonElement>(null);
  const sheetRef = useRef<HTMLDivElement>(null);
  const location = useLocation();
  const isSheetOpen = sheet !== "closed";

  useEffect(() => setSheet("closed"), [location.pathname, mode]);
  useEffect(() => {
    if (!isSheetOpen) return;
    const trigger = moreButton.current;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    sheetRef.current?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setSheet("closed");
      if (event.key !== "Tab" || !sheetRef.current) return;
      const focusable = Array.from(sheetRef.current.querySelectorAll<HTMLElement>('a[href], button:not([disabled])'));
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => { document.body.style.overflow = previousOverflow; document.removeEventListener("keydown", onKeyDown); trigger?.focus(); };
  }, [isSheetOpen]);

  const startDrag = (event: PointerEvent<HTMLButtonElement>) => { dragStart.current = event.clientY; event.currentTarget.setPointerCapture(event.pointerId); };
  const moveDrag = (event: PointerEvent<HTMLButtonElement>) => { if (event.currentTarget.hasPointerCapture(event.pointerId)) setDragOffset(event.clientY - dragStart.current); };
  const endDrag = (event: PointerEvent<HTMLButtonElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    if (dragOffset < -44) setSheet("expanded"); else if (dragOffset > 100 && sheet === "collapsed") setSheet("closed"); else if (dragOffset > 44) setSheet("collapsed");
    setDragOffset(0);
  };
  return (
    <>
      <nav className="game-bottom-nav" aria-label={mode === "mission" ? "Mission navigation" : "Exploration navigation"} data-mode={mode}>
        <div className="game-desktop-nav">{desktopGroups.map((group) => <div className="game-nav-group" key={group.label}><span className="game-nav-group-label">{group.label}</span>{group.items.map((item) => <GameNavLink item={item} key={item.to} />)}</div>)}</div>
        <div className="game-mobile-nav" key={mode}>{primaryItems.map((item) => <GameNavLink item={item} key={item.to} />)}<button ref={moreButton} className={`game-nav-slot game-more-button${sheet !== "closed" ? " active" : ""}`} type="button" aria-haspopup="dialog" aria-expanded={sheet !== "closed"} onClick={() => setSheet("collapsed")}><MoreHorizontal size={21} aria-hidden /><span>{moreLabel}</span></button></div>
      </nav>
      {sheet !== "closed" && <div className="game-sheet-layer"><button className="game-sheet-backdrop" type="button" aria-label={closeLabel} onClick={() => setSheet("closed")} /><div ref={sheetRef} className={`game-more-sheet ${sheet}`} style={{ "--sheet-drag": `${Math.max(-24, dragOffset)}px` } as CSSProperties} role="dialog" aria-modal="true" aria-label={moreLabel} tabIndex={-1}><button className="game-sheet-handle" type="button" aria-label={moreLabel} onPointerDown={startDrag} onPointerMove={moveDrag} onPointerUp={endDrag} onDoubleClick={() => setSheet(sheet === "expanded" ? "collapsed" : "expanded")}><span aria-hidden /></button><div className="game-sheet-header"><strong>{moreLabel}</strong><button type="button" onClick={() => setSheet("closed")} aria-label={closeLabel}>×</button></div><div className="game-sheet-content">{secondaryGroups.map((group) => <section key={group.label}><h2>{group.label}</h2><div className="game-sheet-grid">{group.items.map((item) => <GameNavLink item={item} key={item.to} />)}</div></section>)}</div></div></div>}
    </>
  );
}

function GameNavLink({ item }: { item: GameNavItem }) {
  return <NavLink to={item.to} end={item.end !== false} className={({ isActive }) => `game-nav-slot${isActive ? " active" : ""}`}><item.icon size={21} aria-hidden /><span>{item.label}</span>{item.badge !== undefined && <em>{item.badge}</em>}</NavLink>;
}

export function HudPanel({
  children,
  title,
  eyebrow,
  className = "",
  as: Tag = "section",
}: {
  children: ReactNode;
  title?: ReactNode;
  eyebrow?: ReactNode;
  className?: string;
  as?: "section" | "aside" | "div";
}) {
  return (
    <Tag className={`av-hud-panel frame ${className}`}>
      <span className="frame-brackets" aria-hidden />
      {(title || eyebrow) && (
        <div className="av-hud-head">
          {eyebrow && <span className="av-eyebrow">{eyebrow}</span>}
          {title && <h2>{title}</h2>}
        </div>
      )}
      {children}
    </Tag>
  );
}

const tacticalButtonVariant: Record<
  "primary" | "secondary" | "ghost" | "danger",
  ButtonVariant
> = {
  primary: "tactical",
  secondary: "ghost",
  ghost: "ghost",
  danger: "danger",
};

/**
 * Thin wrapper around the canonical Button primitive (shared/ui/Button.tsx)
 * — kept as its own component so every existing call site's props/className
 * stay unchanged.
 */
export function TacticalButton({
  children,
  to,
  variant = "primary",
  type = "button",
  disabled,
  onClick,
  className = "",
}: {
  children: ReactNode;
  to?: string;
  variant?: "primary" | "secondary" | "ghost" | "danger";
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <Button
      variant={tacticalButtonVariant[variant]}
      to={to}
      type={type}
      disabled={disabled}
      onClick={onClick}
      className={className}
      legacyClassName={`av-tactical-button ${variant}`}
    >
      {children}
    </Button>
  );
}

export function ResourceChip({
  icon,
  label,
  value,
  tone = "green",
}: {
  icon?: ReactNode;
  label?: ReactNode;
  value: ReactNode;
  tone?: "green" | "cyan" | "gold" | "red";
}) {
  return (
    <span className={`av-resource-chip ${tone}`}>
      {icon}
      {label && <span className="av-resource-label">{label}</span>}
      <strong>{value}</strong>
    </span>
  );
}

export function ProgressRing({
  value,
  label,
  size = 58,
  tone = "green",
}: {
  value: number;
  label?: ReactNode;
  size?: number;
  tone?: "green" | "cyan" | "gold" | "red";
}) {
  const clamped = Math.max(0, Math.min(100, Math.round(value)));
  return (
    <span
      className={`av-progress-ring ${tone}`}
      style={{ "--progress": `${clamped}%`, "--ring-size": `${size}px` } as CSSProperties}
      role="img"
      aria-label={`${clamped}%`}
    >
      <span>{label ?? `${clamped}%`}</span>
    </span>
  );
}

export function StatusChip({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "green" | "cyan" | "gold" | "red";
}) {
  return <span className={`av-status-chip ${tone}`}>{children}</span>;
}

export function MissionDossierCard({
  title,
  meta,
  summary,
  progress = 0,
  stats,
  selected,
  action,
  to,
  tone = "green",
}: {
  title: ReactNode;
  meta?: ReactNode;
  summary?: ReactNode;
  progress?: number;
  stats?: ReactNode;
  selected?: boolean;
  action?: ReactNode;
  to?: string;
  tone?: "green" | "cyan" | "gold" | "red";
}) {
  const content = (
    <>
      <div className="av-dossier-top">
        <StatusChip tone={tone}>{meta ?? "OPERATION"}</StatusChip>
        <span className="av-dossier-plus" aria-hidden>
          +
        </span>
      </div>
      <h3>{title}</h3>
      {summary && <p>{summary}</p>}
      <div className="av-dossier-progress">
        <ProgressRing value={progress} tone={tone} />
        <span>
          <small>PROGRESS</small>
          <strong>{Math.round(progress)}%</strong>
        </span>
      </div>
      {stats && <div className="av-dossier-stats">{stats}</div>}
      {action && <div className="av-dossier-action">{action}</div>}
    </>
  );

  if (to) {
    return (
      <Link className={`av-dossier-card ${tone}${selected ? " selected" : ""}`} to={to}>
        {content}
      </Link>
    );
  }
  return (
    <article className={`av-dossier-card ${tone}${selected ? " selected" : ""}`}>
      {content}
    </article>
  );
}

export function TimelineRail({
  items,
}: {
  items: Array<{ time?: ReactNode; title: ReactNode; body?: ReactNode; tone?: "green" | "gold" | "red" | "cyan" }>;
}) {
  return (
    <ol className="av-timeline-rail">
      {items.map((item, index) => (
        <li key={index} className={item.tone ?? "green"}>
          <span className="av-timeline-node" aria-hidden />
          <div>
            {item.time && <time>{item.time}</time>}
            <strong>{item.title}</strong>
            {item.body && <p>{item.body}</p>}
          </div>
        </li>
      ))}
    </ol>
  );
}

export function MissionMapPanel({
  title,
  subtitle,
  markers = 5,
  children,
  to,
  linkLabel,
}: {
  title?: ReactNode;
  subtitle?: ReactNode;
  markers?: number;
  children?: ReactNode;
  /** When set, the whole panel becomes a link to the real map (this preview
   * is atmospheric chrome — the dots aren't real locations — so it should
   * always lead somewhere real rather than being a dead-end decoration). */
  to?: string;
  linkLabel?: string;
}) {
  const content = (
    <>
      <div className="av-map-header">
        <span>
          <Crosshair size={14} aria-hidden />
          {title ?? "TACTICAL MAP"}
        </span>
        {subtitle && <small>{subtitle}</small>}
      </div>
      <div className="av-map-field" aria-hidden={!children}>
        {Array.from({ length: markers }, (_, i) => (
          <span key={i} className={`av-map-marker marker-${i + 1}`}>
            <Circle size={10} aria-hidden />
          </span>
        ))}
        <span className="av-map-origin">
          <ChevronRight size={18} aria-hidden />
        </span>
        {children}
        {to && (
          <span className="av-map-cta">
            <Map size={13} aria-hidden />
            {linkLabel}
          </span>
        )}
      </div>
    </>
  );

  if (to) {
    return (
      <Link className="av-mission-map-panel is-link" to={to} aria-label={linkLabel}>
        {content}
      </Link>
    );
  }
  return <div className="av-mission-map-panel">{content}</div>;
}

export function SelectedMissionBar({
  title,
  meta,
  action,
}: {
  title: ReactNode;
  meta?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="av-selected-mission-bar frame">
      <span className="frame-brackets" aria-hidden />
      <div>
        <span className="av-eyebrow">SELECTED MISSION</span>
        <strong>{title}</strong>
        {meta && <p>{meta}</p>}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}

export function SharedLoadingState({ title = "Synchronizing mission data" }: { title?: string }) {
  return (
    <div className="av-state av-loading-state">
      <Loader2 size={24} className="spin" aria-hidden />
      <strong>{title}</strong>
    </div>
  );
}

export function SharedEmptyState({
  title,
  body,
}: {
  title: ReactNode;
  body?: ReactNode;
}) {
  return (
    <div className="av-state">
      <Radio size={24} aria-hidden />
      <strong>{title}</strong>
      {body && <p>{body}</p>}
    </div>
  );
}

export function SharedErrorState({
  title = "Signal interrupted",
  body,
}: {
  title?: ReactNode;
  body?: ReactNode;
}) {
  return (
    <div className="av-state error">
      <AlertTriangle size={24} aria-hidden />
      <strong>{title}</strong>
      {body && <p>{body}</p>}
    </div>
  );
}

export function DefaultResourceChips({
  wallet,
  energy,
}: {
  wallet?: ReactNode;
  energy?: ReactNode;
}) {
  return (
    <>
      <ResourceChip icon={<Coins size={14} aria-hidden />} value={wallet ?? "—"} tone="gold" />
      <ResourceChip icon={<Zap size={14} aria-hidden />} value={energy ?? "120/120"} tone="green" />
      <ResourceChip icon={<Activity size={14} aria-hidden />} value="API" tone="cyan" />
    </>
  );
}
