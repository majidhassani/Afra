import { useEffect, useRef } from "react";
import { Outlet, useMatch, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
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
  FileText,
  LogOut,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useAuthStore, getRefreshToken } from "@/features/auth/authStore";
import { authApi, gameplayApi } from "@/shared/api/endpoints";
import {
  useEnvironmentTheme,
  environmentFor,
  useWorldModifiers,
} from "@/shared/theme/environment";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { WalletChip, HealthIndicator } from "@/shared/ui/badges";
import { GameBottomNav, GameTopBar, type GameNavItem } from "@/shared/ui/avds";
import { useKeyboardInset } from "@/shared/ui/useKeyboardInset";
import { env } from "@/shared/config/env";
import { useGameplayStatus } from "@/features/game/useGameplayStatus";
import { MissionHUD } from "@/features/game/MissionHUD";
import { GameEffectsLayer } from "@/features/game/GameEffectsLayer";
import { ScrollableTimeline } from "@/features/game/ScrollableTimeline";

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

  // The gameplay-status query is the shell's single source of truth: it
  // themes the backdrop, feeds the always-on HUD, and carries the board art.
  const gameplay = useGameplayStatus(missionId);
  useEnvironmentTheme(
    missionId && gameplay.data ? environmentFor(gameplay.data.mission) : null,
  );
  // Living world: layer weather / night / danger over the biome theme.
  useWorldModifiers(missionId ? gameplay.data?.world_state : null);

  // Scenario board art: use the AI-generated mission board as the scene
  // backdrop; generate it once (best effort) when an active mission has none.
  const qc = useQueryClient();
  const boardRequested = useRef<string | null>(null);
  const board = gameplay.data?.board_art?.mission_board_background;
  const missionActive =
    gameplay.data?.mission.status === "active" ||
    gameplay.data?.mission.status === "ready";
  useEffect(() => {
    if (!missionId || !gameplay.data || !missionActive) return;
    if (board?.url || boardRequested.current === missionId) return;
    boardRequested.current = missionId;
    gameplayApi
      .generateBoard(missionId, "mission_board_background")
      .then(() =>
        qc.invalidateQueries({ queryKey: ["gameplay-status", missionId] }),
      )
      .catch(() => {
        // Board art is a flourish — the themed gradient backdrop remains.
      });
  }, [missionId, gameplay.data, board?.url, missionActive, qc]);

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

  const mainNav: GameNavItem[] = [
    { to: "/app/dashboard", icon: LayoutDashboard, label: t("nav.dashboard") },
    { to: "/app/missions", icon: Rocket, label: t("nav.missions"), end: false },
    { to: "/app/wallet", icon: Wallet, label: t("nav.wallet") },
  ];

  const missionNav: GameNavItem[] = missionId
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
          to: `/app/missions/${missionId}/report`,
          icon: FileText,
          label: t("nav.report"),
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

  const accountNav: GameNavItem[] = [
    { to: "/app/profile", icon: UserRound, label: t("nav.profile") },
    { to: "/app/history", icon: History, label: t("nav.history") },
    { to: "/app/settings", icon: Settings, label: t("nav.settings") },
    ...(env.enableDiagnostics
      ? [{ to: "/app/diagnostics", icon: Activity, label: t("nav.diagnostics") }]
      : []),
  ];

  const tacticalNav: GameNavItem[] = missionId
    ? [
        missionNav[0],
        missionNav[1],
        missionNav[2],
        missionNav[3],
        missionNav[4],
        missionNav[6],
        missionNav[7],
        accountNav[0],
        mainNav[2],
      ].filter((item): item is GameNavItem => Boolean(item))
    : [...mainNav, accountNav[0], accountNav[1], accountNav[2]];

  return (
    <div className="shell">
      {/* Environment-tinted cinematic backdrop + ambient + weather layers. */}
      <div className="env-backdrop" aria-hidden />
      {/* AI-generated scenario board art layered over the gradient backdrop. */}
      {board?.url && (
        <div
          className="board-backdrop"
          style={{ backgroundImage: `url(${board.url})` }}
          aria-hidden
        />
      )}
      <div className="env-atmosphere" aria-hidden />
      <div className="env-weather" aria-hidden />
      <GameTopBar
        brand={t("common.appName")}
        userLabel={user?.display_name}
        resources={
          <>
            <HealthIndicator />
            <WalletChip />
            {env.enableMocks && (
              <span className="chip chip-rare">{t("common.mockMode")}</span>
            )}
          </>
        }
        actions={
          <>
            <LanguageSwitcher />
            <button className="navlink" onClick={logout} title={t("nav.logout")}>
              <LogOut size={16} aria-hidden />
              <span>{t("nav.logout")}</span>
            </button>
          </>
        }
      />

      <main className="main">
        {/* Always-on game HUD: stage, progress, clues, suspect, time, report. */}
        {missionId && gameplay.data && (
          <MissionHUD missionId={missionId} status={gameplay.data} />
        )}
        <Outlet />
      </main>

      {/* Unity-style popups (rewards, stage banners, suspect reveal). */}
      <GameEffectsLayer />

      {/* The everywhere-timeline drawer. */}
      {missionId && (
        <ScrollableTimeline
          missionId={missionId}
          currentTime={gameplay.data?.current_time ?? gameplay.data?.mission.current_time}
        />
      )}

      <GameBottomNav items={tacticalNav} />
    </div>
  );
}
