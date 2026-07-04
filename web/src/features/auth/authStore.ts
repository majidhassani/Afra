import { create } from "zustand";
import type { TokenPair, User } from "@/shared/types/api";

const STORAGE_KEY = "agentverse.session";

interface Session {
  user: User;
  tokens: TokenPair;
}

interface AuthState {
  session: Session | null;
  setSession: (session: Session) => void;
  setTokens: (tokens: TokenPair) => void;
  clear: () => void;
}

function readStoredSession(): Session | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Session;
    if (!parsed?.tokens?.access_token || !parsed?.user?.id) return null;
    if (new Date(parsed.tokens.refresh_expires_at).getTime() < Date.now()) {
      localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function persist(session: Session | null) {
  try {
    if (session) localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    else localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Storage unavailable — session stays in memory only.
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  session: readStoredSession(),
  setSession: (session) => {
    persist(session);
    set({ session });
  },
  setTokens: (tokens) => {
    const current = get().session;
    if (!current) return;
    const next = { ...current, tokens };
    persist(next);
    set({ session: next });
  },
  clear: () => {
    persist(null);
    set({ session: null });
  },
}));

export function getAccessToken(): string | null {
  return useAuthStore.getState().session?.tokens.access_token ?? null;
}

export function getRefreshToken(): string | null {
  return useAuthStore.getState().session?.tokens.refresh_token ?? null;
}
