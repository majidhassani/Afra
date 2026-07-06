export const env = {
  apiBaseUrl:
    (import.meta.env.VITE_API_BASE_URL as string | undefined) ??
    "http://localhost:8080",
  enableMocks:
    String(import.meta.env.VITE_ENABLE_MOCKS ?? "false").toLowerCase() ===
    "true",
  defaultLanguage:
    (import.meta.env.VITE_DEFAULT_LANGUAGE as string | undefined) === "fa"
      ? ("fa" as const)
      : ("en" as const),
  appVersion: (import.meta.env.VITE_APP_VERSION as string | undefined) ?? "dev",
  /**
   * Diagnostics is a developer tool, never part of the shipped game UI.
   * Available in dev builds, or in production only when explicitly enabled.
   */
  enableDiagnostics:
    import.meta.env.DEV ||
    String(import.meta.env.VITE_ENABLE_DIAGNOSTICS ?? "false").toLowerCase() ===
      "true",
  /** Google Maps JS API key; when absent the tactical fallback board renders. */
  googleMapsApiKey:
    (import.meta.env.VITE_GOOGLE_MAPS_API_KEY as string | undefined) ?? "",
};
