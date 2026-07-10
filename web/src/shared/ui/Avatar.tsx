import { useEffect, useState } from "react";
import { Shield, Radio, Skull, User } from "lucide-react";
import { assetUrl } from "@/shared/lib/assetUrl";

/**
 * Avatar renders a character portrait.
 *
 * Privacy rule: this component NEVER receives or renders an avatar prompt.
 * It shows `imageUrl` when the backend provides one; otherwise a premium,
 * deterministic placeholder (category-tinted gradient + initials + tactical
 * frame). It is generation-ready: once the backend returns an `avatar_url`,
 * pass it as `imageUrl` and the portrait renders with no other change.
 */

type Category = "guide" | "field" | "antagonist" | "neutral" | string;
type Size = "sm" | "md" | "lg" | "xl";

const sizePx: Record<Size, number> = { sm: 34, md: 44, lg: 64, xl: 96 };

const categoryIcon: Record<string, typeof User> = {
  guide: Radio,
  field: Shield,
  antagonist: Skull,
  neutral: User,
};

/** Deterministic hue from a string so each character keeps a stable color. */
function hueFromString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) % 360;
  return h;
}

function initials(name: string): string {
  return (
    name
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w[0] ?? "")
      .join("")
      .toUpperCase() || "?"
  );
}

export function Avatar({
  name,
  category,
  size = "md",
  imageUrl,
  version,
  glow,
  shape = "tactical",
}: {
  name: string;
  category?: Category;
  size?: Size;
  imageUrl?: string | null;
  /** Asset version for cache-busting regenerated portraits. */
  version?: number;
  /** Add a subtle active glow (e.g. new dialogue available). */
  glow?: boolean;
  /**
   * "tactical" (default): the premium ~30%-corner portrait frame used
   * everywhere in the app — not a generic circular social-media avatar.
   * "circle": explicit opt-in for genuinely circular/social-profile
   * contexts. Never applied implicitly — a bare global selector never
   * silently overrides this, both shapes only ever come from this prop.
   */
  shape?: "tactical" | "circle";
}) {
  const [broken, setBroken] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const px = sizePx[size];
  const src = assetUrl(imageUrl, version);
  const shapeClass = shape === "circle" ? " avatar-circle" : "";

  // A regenerated portrait changes its cache version. Let the fresh source
  // retry even if the previous URL had failed to load, and re-arm the
  // blur-up transition for the new image.
  useEffect(() => {
    setBroken(false);
    setLoaded(false);
  }, [src]);

  const hue = hueFromString(name);
  const Icon = categoryIcon[category ?? "neutral"] ?? User;
  const placeholder = (
    <span
      className="avatar-ph2"
      style={{
        position: "absolute",
        inset: 0,
        // Two-tone tactical gradient, deterministic per character.
        background: `linear-gradient(140deg, hsl(${hue} 42% 22%), hsl(${(hue + 40) % 360} 38% 12%))`,
        fontSize: px * 0.34,
      }}
      aria-hidden
    >
      <span className="avatar-initials">{initials(name)}</span>
      <Icon className="avatar-cat" size={Math.round(px * 0.28)} aria-hidden />
    </span>
  );

  if (src && !broken) {
    return (
      <span
        className={`avatar avatar-img${shapeClass}${glow ? " avatar-glow" : ""}`}
        style={{ width: px, height: px }}
      >
        {/* Blur-up placeholder: stays visible under the real photo until
            it finishes loading, then fades out — avoids a flash of empty
            space or a broken-image icon while the portrait downloads. */}
        {!loaded && placeholder}
        <img
          src={src}
          alt=""
          width={px}
          height={px}
          loading="lazy"
          decoding="async"
          fetchPriority="low"
          draggable={false}
          style={{
            objectFit: "cover",
            objectPosition: "center 22%",
            opacity: loaded ? 1 : 0,
            transition: "opacity 320ms var(--av-ease-out)",
          }}
          onLoad={() => setLoaded(true)}
          onError={() => setBroken(true)}
        />
      </span>
    );
  }

  return (
    <span
      className={`avatar avatar-ph2${shapeClass}${glow ? " avatar-glow" : ""}`}
      style={{
        width: px,
        height: px,
        background: `linear-gradient(140deg, hsl(${hue} 42% 22%), hsl(${(hue + 40) % 360} 38% 12%))`,
        fontSize: px * 0.34,
      }}
      aria-hidden
    >
      <span className="avatar-initials">{initials(name)}</span>
      <Icon className="avatar-cat" size={Math.round(px * 0.28)} aria-hidden />
    </span>
  );
}
