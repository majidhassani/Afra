import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppShell } from "./AppShell";
import { RequireAuth, RedirectIfAuthed } from "@/features/auth/guards";
import { LoginPage } from "@/features/auth/LoginPage";
import { RegisterPage } from "@/features/auth/RegisterPage";
import { DashboardPage } from "@/features/missions/DashboardPage";
import { MissionsListPage } from "@/features/missions/MissionsListPage";
import { NewMissionPage } from "@/features/missions/NewMissionPage";
import { MissionDashboardPage } from "@/features/missions/MissionDashboardPage";
import { MissionResultPage } from "@/features/missions/MissionResultPage";
import { MapPage } from "@/features/map/MapPage";
import { LocationPage } from "@/features/map/LocationPage";
import { CharactersPage } from "@/features/characters/CharactersPage";
import { CharacterChatPage } from "@/features/characters/CharacterChatPage";
import { CluesPage } from "@/features/clues/CluesPage";
import { ClueDetailPage } from "@/features/clues/ClueDetailPage";
import { JournalPage } from "@/features/journal/JournalPage";
import { EventsPage } from "@/features/events/EventsPage";
import { TimelinePage } from "@/features/missions/TimelinePage";
import { TimePage } from "@/features/time/TimePage";
import { ProfilePage } from "@/features/profile/ProfilePage";
import { WalletPage } from "@/features/wallet/WalletPage";
import { HistoryPage } from "@/features/profile/HistoryPage";
import { SettingsPage } from "@/features/settings/SettingsPage";
import { DiagnosticsPage } from "@/features/diagnostics/DiagnosticsPage";
import { NotFoundPage } from "@/shared/ui/NotFoundPage";
import { env } from "@/shared/config/env";

export const router = createBrowserRouter([
  {
    element: <RedirectIfAuthed />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/register", element: <RegisterPage /> },
    ],
  },
  {
    path: "/app",
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="/app/dashboard" replace /> },
      { path: "dashboard", element: <DashboardPage /> },
      { path: "profile", element: <ProfilePage /> },
      { path: "wallet", element: <WalletPage /> },
      { path: "history", element: <HistoryPage /> },
      { path: "settings", element: <SettingsPage /> },
      // Diagnostics is dev-only; in production the route does not exist.
      ...(env.enableDiagnostics
        ? [{ path: "diagnostics", element: <DiagnosticsPage /> }]
        : []),
      { path: "missions", element: <MissionsListPage /> },
      { path: "missions/new", element: <NewMissionPage /> },
      { path: "missions/:missionId", element: <MissionDashboardPage /> },
      { path: "missions/:missionId/result", element: <MissionResultPage /> },
      { path: "missions/:missionId/map", element: <MapPage /> },
      {
        path: "missions/:missionId/locations/:locationId",
        element: <LocationPage />,
      },
      { path: "missions/:missionId/characters", element: <CharactersPage /> },
      {
        path: "missions/:missionId/characters/:characterId",
        element: <CharacterChatPage />,
      },
      { path: "missions/:missionId/clues", element: <CluesPage /> },
      { path: "missions/:missionId/clues/:clueId", element: <ClueDetailPage /> },
      { path: "missions/:missionId/journal", element: <JournalPage /> },
      { path: "missions/:missionId/timeline", element: <TimelinePage /> },
      { path: "missions/:missionId/events", element: <EventsPage /> },
      { path: "missions/:missionId/time", element: <TimePage /> },
    ],
  },
  { path: "/", element: <Navigate to="/app/dashboard" replace /> },
  { path: "*", element: <NotFoundPage /> },
], {
  future: {
    v7_relativeSplatPath: true,
    v7_fetcherPersist: true,
    v7_normalizeFormMethod: true,
    v7_partialHydration: true,
    v7_skipActionErrorRevalidation: true,
  },
});
