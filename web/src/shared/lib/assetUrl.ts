/**
 * Cache-busting for versioned image assets. When the backend regenerates an
 * avatar/clue image it bumps its version; appending ?ver=N forces the browser
 * to fetch the fresh asset instead of a stale cached one.
 *
 * Data URLs (the current self-contained pipeline) are returned unchanged — the
 * URL already IS the content, so it updates whenever the bytes change and a
 * query string would corrupt it.
 */
export function assetUrl(
  url: string | null | undefined,
  version?: number,
): string | undefined {
  if (!url) return undefined;
  if (url.startsWith("data:")) return url;
  if (!version || version <= 0) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}ver=${version}`;
}
