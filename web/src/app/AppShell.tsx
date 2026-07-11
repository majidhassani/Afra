import { useEffect, useRef } from "react";
import { Outlet, useMatch, useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Home,
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
import { authApi, gameplayApi, missionsApi } from "@/shared/api/endpoints";
import {
  useEnvironmentTheme,
  environmentFor,
  useWorldModifiers,
} from "@/shared/theme/environment";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { Onboarding } from "@/shared/ui/Onboarding";
import { WalletChip, HealthIndicator } from "@/shared/ui/badges";
import { GameBottomNav, GameTopBar, type GameNavGroup, type GameNavItem } from "@/shared/ui/avds";
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
  const missions = useQuery({ queryKey: ["missions"], queryFn: missionsApi.list });
  const activeMission = missions.data?.find((mission) => mission.status === "active" || mission.status === "ready" || mission.status === "generating");
  const routeMissionFinished = gameplay.data?.mission.status === "completed" || gameplay.data?.mission.status === "failed" || gameplay.data?.mission.status === "archived";
  const activeMissionId = routeMissionFinished && activeMission?.id === missionId ? undefined : activeMission?.id;
  const navigationMode = activeMissionId ? "mission" : "exploration";
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

  const explorationPrimary: GameNavItem[] = [
    { to: "/app/dashboard", icon: Home, label: t("nav.hq") },
    { to: "/app/missions", icon: Rocket, label: t("nav.missions"), end: false },
    { to: "/app/missions/new", icon: Sparkles, label: t("nav.ai") },
    { to: "/app/profile", icon: UserRound, label: t("nav.profile") },
  ];

  const missionNav: GameNavItem[] = activeMissionId
    ? [
        { to: `/app/missions/${activeMissionId}`, icon: Rocket, label: t("nav.mission") },
        { to: `/app/missions/${activeMissionId}/map`, icon: Map, label: t("nav.map") },
        {
          to: `/app/missions/${activeMissionId}/ai`,
          icon: Sparkles,
          label: t("nav.ai"),
        },
        {
          to: `/app/missions/${activeMissionId}/characters`,
          icon: Users,
          label: t("nav.characters"),
          end: false,
        },
        {
          to: `/app/missions/${activeMissionId}/clues`,
          icon: Search,
          label: t("nav.evidence"),
          end: false,
        },
        {
          to: `/app/missions/${activeMissionId}/journal`,
          icon: NotebookPen,
          label: t("nav.journal"),
        },
        {
          to: `/app/missions/${activeMissionId}/report`,
          icon: FileText,
          label: t("nav.report"),
        },
        {
          to: `/app/missions/${activeMissionId}/timeline`,
          icon: Radio,
          label: t("nav.timeline"),
        },
        {
          to: `/app/missions/${activeMissionId}/time`,
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

  const missionPrimary = [missionNav[0], missionNav[1], missionNav[3], missionNav[4]].filter((item): item is GameNavItem => Boolean(item));
  const secondaryGroups: GameNavGroup[] = navigationMode === "mission"
    ? [
        { label: t("nav.gameplay"), items: [missionNav[2], missionNav[5], missionNav[6], missionNav[7], missionNav[8]].filter((item): item is GameNavItem => Boolean(item)) },
        { label: t("nav.utility"), items: [mainNavWallet(), accountNav[0], accountNav[1]] },
        { label: t("nav.settingsGroup"), items: accountNav.slice(2) },
      ]
    : [
        { label: t("nav.utility"), items: [mainNavWallet(), accountNav[1]] },
        { label: t("nav.settingsGroup"), items: accountNav.slice(2) },
      ];
  const desktopGroups: GameNavGroup[] = navigationMode === "mission"
    ? [
        { label: t("nav.gameplay"), items: [...missionPrimary, missionNav[2], missionNav[5], missionNav[6], missionNav[7], missionNav[8]].filter((item): item is GameNavItem => Boolean(item)) },
        ...secondaryGroups.slice(1),
      ]
    : [{ label: t("nav.gameplay"), items: explorationPrimary }, ...secondaryGroups];

  function mainNavWallet(): GameNavItem {
    return { to: "/app/wallet", icon: Wallet, label: t("nav.wallet") };
  }

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

      <GameBottomNav primaryItems={navigationMode === "mission" ? missionPrimary : explorationPrimary} secondaryGroups={secondaryGroups} desktopGroups={desktopGroups} moreLabel={t("nav.more")} closeLabel={t("common.close")} mode={navigationMode} />

      {/* First-time walkthrough — only outside an active mission, so it
          never interrupts real gameplay context (e.g. a refresh mid-clue). */}
      <Onboarding eligible={!missionId} />
    </div>
  );
}
