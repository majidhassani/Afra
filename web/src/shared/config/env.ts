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
};
