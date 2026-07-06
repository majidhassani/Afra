import { env } from "@/shared/config/env";
import {
  getRefreshToken,
  getAccessToken,
  useAuthStore,
} from "@/features/auth/authStore";
import type { APIErrorBody, TokenPair } from "@/shared/types/api";
import { mockRequest } from "./mock/mockClient";
import { localeHeaders } from "@/shared/i18n/locale";

/** Uniform error thrown for every failed API call. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }

  get isNetwork() {
    return this.code === "network_error";
  }
  get isUnauthorized() {
    return this.status === 401;
  }
}

/** Diagnostics: last API error observed anywhere in the app. */
export const lastApiError: { current: ApiError | null } = { current: null };

interface Envelope<T> {
  data?: T;
  error?: APIErrorBody;
}

export interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  signal?: AbortSignal;
  /** Skip Authorization header (auth endpoints). */
  anonymous?: boolean;
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    const refreshToken = getRefreshToken();
    if (!refreshToken) return false;
    try {
      const res = await fetch(`${env.apiBaseUrl}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });
      if (!res.ok) return false;
      const envl = (await res.json()) as Envelope<{ tokens: TokenPair }>;
      if (!envl.data?.tokens) return false;
      useAuthStore.getState().setTokens(envl.data.tokens);
      return true;
    } catch {
      return false;
    }
  })();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

async function rawRequest<T>(path: string, opts: RequestOptions): Promise<T> {
  // Every request advertises the selected UI language so the backend answers
  // in it (Accept-Language / X-App-Language / X-UI-Direction).
  const headers: Record<string, string> = { ...localeHeaders() };
  if (opts.body !== undefined) headers["Content-Type"] = "application/json";
  if (!opts.anonymous) {
    const token = getAccessToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  let res: Response;
  try {
    res = await fetch(`${env.apiBaseUrl}${path}`, {
      method: opts.method ?? "GET",
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: opts.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") throw err;
    throw new ApiError("network_error", "network request failed", 0);
  }

  if (res.status === 204) return undefined as T;

  let envelope: Envelope<T>;
  try {
    envelope = (await res.json()) as Envelope<T>;
  } catch {
    throw new ApiError("invalid_response", "invalid server response", res.status);
  }

  if (!res.ok || envelope.error) {
    throw new ApiError(
      envelope.error?.code ?? `http_${res.status}`,
      envelope.error?.message ?? res.statusText,
      res.status,
    );
  }

  return envelope.data as T;
}

/**
 * Perform an API request against the backend (or the mock adapter when
 * VITE_ENABLE_MOCKS=true). Handles the {data}/{error} envelope, bearer token
 * injection, and a single transparent refresh+retry on 401.
 */
export async function apiRequest<T>(
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  if (env.enableMocks) {
    return mockRequest<T>(path, opts);
  }
  try {
    return await rawRequest<T>(path, opts);
  } catch (err) {
    if (err instanceof ApiError && err.isUnauthorized && !opts.anonymous) {
      const refreshed = await tryRefresh();
      if (refreshed) {
        return rawRequest<T>(path, opts);
      }
      useAuthStore.getState().clear();
      lastApiError.current = err;
      throw err;
    }
    if (err instanceof ApiError) lastApiError.current = err;
    throw err;
  }
}

/** Map an ApiError to a translation key for user display. */
export function errorKey(err: unknown):
  | "error.network"
  | "error.unauthorized"
  | "error.insufficient_balance"
  | "error.mission_generating"
  | "error.not_found"
  | "error.validation"
  | "error.conflict"
  | "error.rate_limited"
  | "error.server" {
  if (!(err instanceof ApiError)) return "error.server";
  if (err.isNetwork) return "error.network";
  if (err.status === 401) return "error.unauthorized";
  if (err.code.includes("insufficient") || err.code === "insufficient_balance")
    return "error.insufficient_balance";
  if (err.code.includes("generating")) return "error.mission_generating";
  if (err.status === 404) return "error.not_found";
  if (err.status === 400) return "error.validation";
  if (err.status === 409) return "error.conflict";
  if (err.status === 429) return "error.rate_limited";
  return "error.server";
}
