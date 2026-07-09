import type { CSSProperties, ReactNode } from "react";
import { Link, NavLink } from "react-router-dom";
import {
  Activity,
  AlertTriangle,
  ChevronRight,
  Circle,
  Coins,
  Crosshair,
  Loader2,
  Radio,
  type LucideIcon,
  Zap,
} from "lucide-react";

export interface GameNavItem {
  to: string;
  icon: LucideIcon;
  label: string;
  end?: boolean;
  badge?: string | number;
}

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

export function GameBottomNav({ items }: { items: GameNavItem[] }) {
  return (
    <nav className="game-bottom-nav" aria-label="Mission navigation">
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.end !== false}
          className={({ isActive }) => `game-nav-slot${isActive ? " active" : ""}`}
        >
          <item.icon size={21} aria-hidden />
          <span>{item.label}</span>
          {item.badge !== undefined && <em>{item.badge}</em>}
        </NavLink>
      ))}
    </nav>
  );
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
  const cls = `av-tactical-button ${variant} ${className}`;
  if (to) {
    return (
      <Link className={cls} to={to}>
        {children}
      </Link>
    );
  }
  return (
    <button className={cls} type={type} disabled={disabled} onClick={onClick}>
      {children}
    </button>
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
}: {
  title?: ReactNode;
  subtitle?: ReactNode;
  markers?: number;
  children?: ReactNode;
}) {
  return (
    <div className="av-mission-map-panel">
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
      </div>
    </div>
  );
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
