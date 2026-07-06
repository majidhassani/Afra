import { env } from "@/shared/config/env";

/**
 * Google Maps JS API loader. The key comes from VITE_GOOGLE_MAPS_API_KEY;
 * when absent, callers must render the tactical fallback board instead.
 */

let loader: Promise<typeof google.maps> | null = null;

export function hasGoogleMapsKey(): boolean {
  return env.googleMapsApiKey.length > 0;
}

export function loadGoogleMaps(): Promise<typeof google.maps> {
  if (!hasGoogleMapsKey()) {
    return Promise.reject(new Error("google maps api key missing"));
  }
  if (loader) return loader;
  loader = new Promise((resolve, reject) => {
    if (typeof google !== "undefined" && google.maps?.Map) {
      resolve(google.maps);
      return;
    }
    const callbackName = "__agentverseMapsReady";
    (window as unknown as Record<string, unknown>)[callbackName] = () => {
      resolve(google.maps);
      delete (window as unknown as Record<string, unknown>)[callbackName];
    };
    const script = document.createElement("script");
    script.src =
      `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(env.googleMapsApiKey)}` +
      `&callback=${callbackName}&v=weekly&loading=async`;
    script.async = true;
    script.onerror = () => {
      loader = null;
      reject(new Error("failed to load google maps"));
    };
    document.head.appendChild(script);
  });
  return loader;
}

/** Dark cinematic map style matching the AVDS palette. */
export const AVDS_MAP_STYLE: google.maps.MapTypeStyle[] = [
  { elementType: "geometry", stylers: [{ color: "#12151b" }] },
  { elementType: "labels.text.fill", stylers: [{ color: "#7d7a72" }] },
  { elementType: "labels.text.stroke", stylers: [{ color: "#090b0f" }] },
  { featureType: "administrative", elementType: "geometry", stylers: [{ color: "#2c3036" }] },
  { featureType: "poi", stylers: [{ visibility: "off" }] },
  { featureType: "poi.park", elementType: "geometry", stylers: [{ color: "#101812" }] },
  { featureType: "road", elementType: "geometry", stylers: [{ color: "#1a1f27" }] },
  { featureType: "road", elementType: "geometry.stroke", stylers: [{ color: "#232935" }] },
  { featureType: "road", elementType: "labels", stylers: [{ visibility: "off" }] },
  { featureType: "road.highway", elementType: "geometry", stylers: [{ color: "#232935" }] },
  { featureType: "transit", stylers: [{ visibility: "off" }] },
  { featureType: "water", elementType: "geometry", stylers: [{ color: "#0b1218" }] },
  { featureType: "water", elementType: "labels.text.fill", stylers: [{ color: "#3d5a63" }] },
];
