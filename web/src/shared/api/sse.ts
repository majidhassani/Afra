import { useEffect, useRef, useState } from "react";
import { env } from "@/shared/config/env";
import { getAccessToken } from "@/features/auth/authStore";
import { missionsApi } from "./endpoints";
import type { MissionEvent } from "@/shared/types/api";

export type StreamStatus = "live" | "polling" | "disconnected";

interface StreamOptions {
  /** Called for every event received (SSE frame or newly-polled event). */
  onEvent?: (event: MissionEvent) => void;
  /** Polling interval used in fallback mode. */
  pollMs?: number;
  enabled?: boolean;
}

/**
 * Live mission event stream.
 *
 * EventSource cannot send Authorization headers, so we stream the SSE
 * endpoint via fetch + ReadableStream. If the stream cannot be established
 * (or in mock mode) we fall back to polling GET /events.
 */
export function useMissionStream(
  missionId: string | undefined,
  { onEvent, pollMs = 5000, enabled = true }: StreamOptions = {},
): StreamStatus {
  const [status, setStatus] = useState<StreamStatus>("disconnected");
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!missionId || !enabled) {
      setStatus("disconnected");
      return;
    }

    let cancelled = false;
    let pollTimer: ReturnType<typeof setInterval> | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    const controller = new AbortController();
    const seen = new Set<string>();

    const emit = (event: MissionEvent) => {
      if (event.id && seen.has(event.id)) return;
      if (event.id) seen.add(event.id);
      onEventRef.current?.(event);
    };

    const startPolling = () => {
      if (cancelled || pollTimer) return;
      setStatus("polling");
      const poll = async () => {
        try {
          const events = await missionsApi.events(missionId, 30);
          if (cancelled) return;
          for (const ev of [...events].reverse()) emit(ev);
        } catch {
          if (!cancelled) setStatus("disconnected");
        }
      };
      void poll();
      pollTimer = setInterval(poll, pollMs);
    };

    const startStream = async (attempt: number) => {
      if (cancelled) return;
      if (env.enableMocks) {
        startPolling();
        return;
      }
      try {
        const token = getAccessToken();
        const res = await fetch(
          `${env.apiBaseUrl}/api/v1/missions/${missionId}/stream`,
          {
            headers: token ? { Authorization: `Bearer ${token}` } : {},
            signal: controller.signal,
          },
        );
        if (!res.ok || !res.body) throw new Error(`stream ${res.status}`);
        setStatus("live");

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const frames = buffer.split("\n\n");
          buffer = frames.pop() ?? "";
          for (const frame of frames) {
            const dataLines = frame
              .split("\n")
              .filter((l) => l.startsWith("data:"))
              .map((l) => l.slice(5).trim());
            if (dataLines.length === 0) continue;
            try {
              emit(JSON.parse(dataLines.join("\n")) as MissionEvent);
            } catch {
              // Non-JSON keepalive frame — ignore.
            }
          }
        }
        throw new Error("stream ended");
      } catch (err) {
        if (cancelled || (err instanceof DOMException && err.name === "AbortError"))
          return;
        if (attempt < 3) {
          setStatus("disconnected");
          retryTimer = setTimeout(
            () => void startStream(attempt + 1),
            1000 * 2 ** attempt,
          );
        } else {
          startPolling();
        }
      }
    };

    void startStream(0);

    return () => {
      cancelled = true;
      controller.abort();
      if (pollTimer) clearInterval(pollTimer);
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [missionId, enabled, pollMs]);

  return status;
}
