import { useState } from "react";
import { Shield, Radio, Skull, User } from "lucide-react";

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
  glow,
}: {
  name: string;
  category?: Category;
  size?: Size;
  imageUrl?: string | null;
  /** Add a subtle active glow (e.g. new dialogue available). */
  glow?: boolean;
}) {
  const [broken, setBroken] = useState(false);
  const px = sizePx[size];

  if (imageUrl && !broken) {
    return (
      <span
        className={`avatar avatar-img${glow ? " avatar-glow" : ""}`}
        style={{ width: px, height: px }}
      >
        <img src={imageUrl} alt="" onError={() => setBroken(true)} />
      </span>
    );
  }

  const hue = hueFromString(name);
  const Icon = categoryIcon[category ?? "neutral"] ?? User;
  return (
    <span
      className={`avatar avatar-ph2${glow ? " avatar-glow" : ""}`}
      style={{
        width: px,
        height: px,
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
}
