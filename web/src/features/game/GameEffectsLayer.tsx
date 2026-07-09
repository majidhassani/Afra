import { useEffect } from "react";
import { Award, Clock3, MapPin, Radio, Sparkles, UserRound } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { Avatar } from "@/shared/ui/Avatar";
import { useGameEvents, type GameEvent } from "./gameEvents";

/**
 * Fixed overlay that renders queued game events as Unity-style popups:
 * STAGE COMPLETE banners, reward toasts, the SUSPECT IDENTIFIED reveal,
 * unlock notices, world events, and time-passed toasts. Mounted once in the
 * AppShell so effects appear on every screen.
 */
export function GameEffectsLayer() {
  const queue = useGameEvents((s) => s.queue);
  if (queue.length === 0) return null;
  // The suspect reveal is modal; everything else stacks as toasts.
  const reveal = queue.find((e) => e.kind === "suspect");
  const toasts = queue.filter((e) => e.kind !== "suspect").slice(0, 4);
  return (
    <div className="game-effects" aria-live="polite">
      {reveal && <SuspectReveal event={reveal} />}
      <div className="game-toasts">
        {toasts.map((ev) => (
          <GameToast key={ev.id} event={ev} />
        ))}
      </div>
    </div>
  );
}

function SuspectReveal({ event }: { event: GameEvent & { kind: "suspect" } }) {
  const { t } = useI18n();
  const dismiss = useGameEvents((s) => s.dismiss);
  return (
    <div className="suspect-reveal" role="dialog" aria-modal="true">
      <div className="suspect-reveal-card">
        <div className="suspect-reveal-flash" aria-hidden />
        <span className="suspect-reveal-kicker">
          <UserRound size={16} aria-hidden /> {t("game.suspectIdentified")}
        </span>
        {event.suspect && (
          <>
            <Avatar
              imageUrl={event.suspect.avatar_url}
              name={event.suspect.name}
              size="lg"
              glow
            />
            <div className="suspect-reveal-name">{event.suspect.name}</div>
            <div className="suspect-reveal-role">{event.suspect.role}</div>
          </>
        )}
        <button className="game-btn game-btn-primary" onClick={() => dismiss(event.id)}>
          {t("game.proceed")}
        </button>
      </div>
    </div>
  );
}

function GameToast({ event }: { event: GameEvent }) {
  const { t } = useI18n();
  const dismiss = useGameEvents((s) => s.dismiss);

  // Toasts auto-dismiss; banners linger a little longer.
  useEffect(() => {
    const ttl = event.kind === "stage_complete" ? 6000 : 4500;
    const timer = setTimeout(() => dismiss(event.id), ttl);
    return () => clearTimeout(timer);
  }, [event.id, event.kind, dismiss]);

  switch (event.kind) {
    case "stage_complete":
      return (
        <button className="game-toast stage-banner" onClick={() => dismiss(event.id)}>
          <Sparkles size={16} aria-hidden />
          <span>
            <strong>{t("game.stageComplete")}</strong> — {event.stage.title}
          </span>
        </button>
      );
    case "stage_unlocked":
      return (
        <button className="game-toast" onClick={() => dismiss(event.id)}>
          <Radio size={15} aria-hidden />
          <span>{t("game.stageUnlocked", { title: event.stage.title })}</span>
        </button>
      );
    case "reward":
      return (
        <button className="game-toast reward-toast" onClick={() => dismiss(event.id)}>
          <Award size={15} aria-hidden />
          <span>
            {event.xp > 0 && t("game.rewardXp", { xp: event.xp })}{" "}
            {event.coins > 0 && t("game.rewardCoins", { coins: event.coins })}
            {event.badge ? ` · ${event.badge}` : ""}
          </span>
        </button>
      );
    case "location_unlocked":
      return (
        <button className="game-toast" onClick={() => dismiss(event.id)}>
          <MapPin size={15} aria-hidden />
          <span>{t("game.locationUnlocked", { name: event.name })}</span>
        </button>
      );
    case "world_event":
      return (
        <button className="game-toast world-toast" onClick={() => dismiss(event.id)}>
          <Radio size={15} aria-hidden />
          <span>
            <strong>{t("game.worldEvent")}:</strong> {event.title}
          </span>
        </button>
      );
    case "time":
      return (
        <button className="game-toast time-toast" onClick={() => dismiss(event.id)}>
          <Clock3 size={15} aria-hidden />
          <span>
            {t("game.timePassed", { minutes: event.minutes, time: event.newTime })}
          </span>
        </button>
      );
    default:
      return null;
  }
}
