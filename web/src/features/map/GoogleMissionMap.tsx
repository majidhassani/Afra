import { useEffect, useRef, useState } from "react";
import { loadGoogleMaps, AVDS_MAP_STYLE } from "@/shared/lib/googleMaps";
import type { MapView, Marker } from "@/shared/types/api";

/**
 * GoogleMissionMap — the real Google Maps gameplay surface. Mission markers
 * render as HTML overlays (same .marker-pin visual system as the fallback
 * board) so every marker state — recommended pulse, new-clue badge,
 * character badge, high-risk, completed, locked — reads identically on the
 * real map.
 */

const MARKER_ICONS: Record<string, string> = {
  locked:
    '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>',
  visited:
    '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21.801 10A10 10 0 1 1 17 3.335"/><path d="m9 11 3 3L22 4"/></svg>',
  pin:
    '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 14.993 4 10a8 8 0 0 1 16 0"/><circle cx="12" cy="10" r="3"/></svg>',
};

function markerClasses(marker: Marker, selected: boolean): string {
  return [
    "marker-pin",
    "gmap-pin",
    marker.is_locked ? "locked" : "",
    marker.status === "visited" ? "visited" : "",
    selected ? "selected" : "",
    marker.recommended && !marker.is_locked ? "recommended pulse" : "",
    marker.risk_level >= 60 || marker.objective_status === "high_risk"
      ? "highrisk"
      : "",
    marker.has_new_clue && !marker.is_locked ? "pulse" : "",
  ]
    .filter(Boolean)
    .join(" ");
}

function markerHTML(marker: Marker): string {
  const icon = marker.is_locked
    ? MARKER_ICONS.locked
    : marker.status === "visited"
      ? MARKER_ICONS.visited
      : MARKER_ICONS.pin;
  const flags = [
    marker.recommended ? '<span class="marker-flag recommended"></span>' : "",
    marker.has_new_clue ? '<span class="marker-flag clue"></span>' : "",
    marker.has_character ? '<span class="marker-flag character"></span>' : "",
  ].join("");
  const label = document.createElement("span");
  label.textContent = marker.name;
  return `<span class="marker-dot">${icon}${flags}</span><span class="marker-label">${label.innerHTML}</span>`;
}

export function GoogleMissionMap({
  view,
  selectedId,
  onSelect,
  onUnavailable,
}: {
  view: MapView;
  selectedId: string | null;
  onSelect: (id: string) => void;
  onUnavailable: () => void;
}) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<google.maps.Map | null>(null);
  const overlaysRef = useRef<Map<string, HTMLButtonElement>>(new Map());
  const overlayViewRef = useRef<google.maps.OverlayView | null>(null);
  const [ready, setReady] = useState(false);
  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  // Boot the map once.
  useEffect(() => {
    let cancelled = false;
    loadGoogleMaps()
      .then((maps) => {
        if (cancelled || !containerRef.current) return;
        const map = new maps.Map(containerRef.current, {
          center: { lat: view.center.lat, lng: view.center.lng },
          zoom: view.zoom || 13,
          styles: AVDS_MAP_STYLE,
          disableDefaultUI: true,
          zoomControl: true,
          gestureHandling: "greedy",
          backgroundColor: "#090b0f",
          clickableIcons: false,
        });
        mapRef.current = map;
        setReady(true);
      })
      .catch(() => {
        if (!cancelled) onUnavailable();
      });
    return () => {
      cancelled = true;
    };
    // The map is created once; marker/center updates are handled below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Render mission markers as an HTML overlay layer.
  useEffect(() => {
    const map = mapRef.current;
    if (!ready || !map) return;

    class MissionOverlay extends google.maps.OverlayView {
      onAdd() {
        const pane = this.getPanes()?.overlayMouseTarget;
        if (!pane) return;
        for (const marker of view.locations) {
          const el = document.createElement("button");
          el.type = "button";
          el.className = markerClasses(marker, marker.id === selectedId);
          el.innerHTML = markerHTML(marker);
          el.setAttribute("aria-label", marker.name);
          el.addEventListener("click", (e) => {
            e.stopPropagation();
            onSelectRef.current(marker.id);
          });
          pane.appendChild(el);
          overlaysRef.current.set(marker.id, el);
        }
      }
      draw() {
        const projection = this.getProjection();
        if (!projection) return;
        for (const marker of view.locations) {
          const el = overlaysRef.current.get(marker.id);
          if (!el) continue;
          const point = projection.fromLatLngToDivPixel(
            new google.maps.LatLng(marker.lat, marker.lng),
          );
          if (!point) continue;
          el.style.left = `${point.x}px`;
          el.style.top = `${point.y}px`;
        }
      }
      onRemove() {
        for (const el of overlaysRef.current.values()) el.remove();
        overlaysRef.current.clear();
      }
    }

    const overlay = new MissionOverlay();
    overlay.setMap(map);
    overlayViewRef.current = overlay;

    // Fit all markers in view once.
    if (view.locations.length > 1) {
      const bounds = new google.maps.LatLngBounds();
      for (const m of view.locations) bounds.extend({ lat: m.lat, lng: m.lng });
      map.fitBounds(bounds, 64);
    }

    return () => {
      overlay.setMap(null);
      overlayViewRef.current = null;
    };
    // Rebuild the overlay layer when the marker set changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, view.locations]);

  // Reflect selection without rebuilding the layer.
  useEffect(() => {
    for (const marker of view.locations) {
      const el = overlaysRef.current.get(marker.id);
      if (el) el.className = markerClasses(marker, marker.id === selectedId);
    }
    if (selectedId && mapRef.current) {
      const marker = view.locations.find((m) => m.id === selectedId);
      if (marker) mapRef.current.panTo({ lat: marker.lat, lng: marker.lng });
    }
  }, [selectedId, view.locations]);

  return <div ref={containerRef} className="gmap-canvas" />;
}
