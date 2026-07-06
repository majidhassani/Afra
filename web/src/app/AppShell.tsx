import { NavLink, Outlet, useMatch, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  LayoutDashboard,
  Rocket,
  Wallet,
  UserRound,
  History,
  Settings,
  Activity,
  Map,
  Users,
  Search,
  NotebookPen,
  Radio,
  Clock3,
  Sparkles,
  LogOut,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useAuthStore, getRefreshToken } from "@/features/auth/authStore";
import { authApi, missionsApi } from "@/shared/api/endpoints";
import { useEnvironmentTheme, environmentFor } from "@/shared/theme/environment";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { WalletChip, HealthIndicator } from "@/shared/ui/badges";
import { useKeyboardInset } from "@/shared/ui/useKeyboardInset";
import { env } from "@/shared/config/env";

export function AppShell() {
  const { t } = useI18n();
  useKeyboardInset();
  const navigate = useNavigate();
  const clear = useAuthStore((s) => s.clear);
  const user = useAuthStore((s) => s.session?.user);
  const missionMatch = useMatch("/app/missions/:missionId/*");
  const missionId =
    missionMatch?.params.missionId === "new"
      ? undefined
      : missionMatch?.params.missionId;

  // Tint the whole shell (backdrop + accents) to the active mission's
  // environment, so every mission sub-page shares the biome look. Reuses the
  // cached dashboard query so it costs no extra request.
  const missionEnv = useQuery({
    queryKey: ["mission", missionId],
    queryFn: () => missionsApi.dashboard(missionId!),
    enabled: !!missionId,
  });
  useEnvironmentTheme(
    missionId && missionEnv.data ? environmentFor(missionEnv.data.mission) : null,
  );

  const logout = async () => {
    const refreshToken = getRefreshToken();
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch {
      // Server-side logout is best-effort; local session is cleared anyway.
    }
    clear();
    navigate("/login");
  };

  interface NavEntry {
    to: string;
    icon: typeof Map;
    label: string;
    end?: boolean;
  }

  const mainNav: NavEntry[] = [
    { to: "/app/dashboard", icon: LayoutDashboard, label: t("nav.dashboard") },
    { to: "/app/missions", icon: Rocket, label: t("nav.missions"), end: false },
    { to: "/app/wallet", icon: Wallet, label: t("nav.wallet") },
  ];

  const missionNav: NavEntry[] = missionId
    ? [
        { to: `/app/missions/${missionId}`, icon: Rocket, label: t("nav.overview") },
        { to: `/app/missions/${missionId}/map`, icon: Map, label: t("nav.map") },
        {
          to: `/app/missions/${missionId}/ai`,
          icon: Sparkles,
          label: t("nav.ai"),
        },
        {
          to: `/app/missions/${missionId}/characters`,
          icon: Users,
          label: t("nav.characters"),
          end: false,
        },
        {
          to: `/app/missions/${missionId}/clues`,
          icon: Search,
          label: t("nav.clues"),
          end: false,
        },
        {
          to: `/app/missions/${missionId}/journal`,
          icon: NotebookPen,
          label: t("nav.journal"),
        },
        {
          to: `/app/missions/${missionId}/timeline`,
          icon: Radio,
          label: t("nav.timeline"),
        },
        {
          to: `/app/missions/${missionId}/time`,
          icon: Clock3,
          label: t("nav.time"),
        },
      ]
    : [];

  const accountNav: NavEntry[] = [
    { to: "/app/profile", icon: UserRound, label: t("nav.profile") },
    { to: "/app/history", icon: History, label: t("nav.history") },
    { to: "/app/settings", icon: Settings, label: t("nav.settings") },
    ...(env.enableDiagnostics
      ? [{ to: "/app/diagnostics", icon: Activity, label: t("nav.diagnostics") }]
      : []),
  ];

  // Mobile bottom nav follows the game layout: Mission | Map | AI | Clues |
  // Profile in a mission; the lobby essentials otherwise. Wallet stays in HUD.
  const mobileNav = missionId
    ? [
        missionNav[0], // Mission
        missionNav[1], // Map
        missionNav[2], // AI
        missionNav[4], // Clues
        accountNav[0], // Profile
      ]
    : [...mainNav, accountNav[0], accountNav[1]];

  return (
    <div className="shell">
      {/* Environment-tinted cinematic backdrop + ambient layer (via [data-env]). */}
      <div className="env-backdrop" aria-hidden />
      <div className="env-atmosphere" aria-hidden />
      <nav className="sidenav" aria-label="Main">
        <div className="sidenav-brand">{t("common.appName")}</div>
        {mainNav.map((item) => (
          <NavItem key={item.to} {...item} />
        ))}
        {missionNav.length > 0 && (
          <>
            <div className="sidenav-section">{t("nav.mission")}</div>
            {missionNav.map((item) => (
              <NavItem key={item.to} {...item} />
            ))}
          </>
        )}
        <div className="sidenav-section">{t("nav.account")}</div>
        {accountNav.map((item) => (
          <NavItem key={item.to} {...item} />
        ))}
        <div style={{ flex: 1 }} />
        <button className="navlink" onClick={logout}>
          <LogOut size={16} aria-hidden />
          {t("nav.logout")}
        </button>
      </nav>

      <header className="topbar">
        <div className="row" style={{ minWidth: 0 }}>
          <span className="muted" style={{ fontSize: 13 }}>
            {user?.display_name}
          </span>
          {env.enableMocks && (
            <span className="chip chip-rare">{t("common.mockMode")}</span>
          )}
        </div>
        <div className="row">
          <HealthIndicator />
          <WalletChip />
          <LanguageSwitcher />
        </div>
      </header>

      <main className="main">
        <Outlet />
      </main>

      <nav className="mobilenav" aria-label="Mobile">
        {mobileNav.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end !== false}
            className={({ isActive }) => (isActive ? "active" : "")}
          >
            <item.icon size={18} aria-hidden />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  );
}

function NavItem({
  to,
  icon: Icon,
  label,
  end,
}: {
  to: string;
  icon: typeof Map;
  label: string;
  end?: boolean;
}) {
  return (
    <NavLink
      to={to}
      end={end !== false}
      className={({ isActive }) => `navlink${isActive ? " active" : ""}`}
    >
      <Icon size={16} aria-hidden />
      {label}
    </NavLink>
  );
}
