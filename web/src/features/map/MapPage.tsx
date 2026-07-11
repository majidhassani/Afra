import { useEffect, useMemo, useRef, useState, type PointerEvent } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  MapPin,
  Lock,
  CheckCircle2,
  ExternalLink,
  ShieldAlert,
  Search,
  MessageCircle,
  Sparkles,
  ChevronDown,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { mapApi } from "@/shared/api/endpoints";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import type { Marker } from "@/shared/types/api";
import { Avatar } from "@/shared/ui/Avatar";
import { GameButton } from "@/shared/ui/game";
import { hasGoogleMapsKey } from "@/shared/lib/googleMaps";
import { GoogleMissionMap } from "./GoogleMissionMap";

/**
 * Mission map. With a Google Maps key the real map renders (markers as HTML
 * overlays with full game state); without one the tactical fallback board
 * takes over. Marker tap opens the side panel (desktop) or the bottom sheet
 * (mobile) with location intel and actions.
 */
export function MapPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [mapsUnavailable, setMapsUnavailable] = useState(false);
  const [sheetExpanded, setSheetExpanded] = useState(false);
  const sheetHandleRef = useRef<HTMLButtonElement>(null);
  const dragStartY = useRef<number | null>(null);
  const useRealMap = hasGoogleMapsKey() && !mapsUnavailable;

  const map = useQuery({
    queryKey: ["mission", missionId, "map"],
    queryFn: () => mapApi.view(missionId!),
    enabled: !!missionId,
  });

  const selected =
    map.data?.locations.find((m) => m.id === selectedId) ?? null;

  // Selecting a marker on mobile should surface its detail immediately —
  // the sheet expands automatically rather than requiring a second tap.
  useEffect(() => {
    if (selected) {
      setSheetExpanded(true);
      sheetHandleRef.current?.focus({ preventScroll: true });
    }
  }, [selected]);

  useEffect(() => {
    if (!sheetExpanded) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setSheetExpanded(false);
        sheetHandleRef.current?.focus({ preventScroll: true });
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [sheetExpanded]);

  const startSheetDrag = (event: PointerEvent<HTMLButtonElement>) => {
    dragStartY.current = event.clientY;
    event.currentTarget.setPointerCapture(event.pointerId);
  };
  const moveSheetDrag = (event: PointerEvent<HTMLButtonElement>) => {
    if (dragStartY.current === null) return;
    const distance = event.clientY - dragStartY.current;
    if (distance < -32) setSheetExpanded(true);
    if (distance > 32) setSheetExpanded(false);
  };
  const endSheetDrag = (event: PointerEvent<HTMLButtonElement>) => {
    dragStartY.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  };

  return (
    <div className="map-layout">
      <h1 className="visually-hidden">{t("map.title")}</h1>
      <div
        className="map-board-wrap"
        role="application"
        aria-label={t("map.title")}
      >
        {map.isPending && <SkeletonRows rows={5} />}
        {map.isError && (
          <ErrorState error={map.error} onRetry={() => map.refetch()} />
        )}
        {map.data && useRealMap && (
          <GoogleMissionMap
            view={map.data}
            selectedId={selectedId}
            onSelect={setSelectedId}
            onUnavailable={() => setMapsUnavailable(true)}
          />
        )}
        {map.data && !useRealMap && (
          <FallbackBoard
            markers={map.data.locations}
            selectedId={selectedId}
            onSelect={setSelectedId}
          />
        )}
        {import.meta.env.DEV && !hasGoogleMapsKey() && (
          <div className="map-dev-warning">{t("map.devNoKey")}</div>
        )}
      </div>

      {/* Desktop: a persistent side panel (unchanged). Mobile: the same
          markup becomes a bottom sheet — see .map-side's mobile rule in
          app.css. `sheet-collapsed`/`sheet-expanded` only affect layout
          below the map-board breakpoint; on desktop they're inert. */}
      <aside
        className={`map-side ${sheetExpanded ? "sheet-expanded" : "sheet-collapsed"}`}
        aria-label={t("map.selectLocation")}
      >
        <button
          ref={sheetHandleRef}
          type="button"
          className="map-side-handle"
          onClick={() => setSheetExpanded((v) => !v)}
          aria-expanded={sheetExpanded}
          aria-controls="map-side-body"
          onPointerDown={startSheetDrag}
          onPointerMove={moveSheetDrag}
          onPointerUp={endSheetDrag}
          onPointerCancel={endSheetDrag}
        >
          <span className="map-side-grip" aria-hidden />
          <span className="map-side-handle-label">
            {selected ? selected.name : t("map.selectLocation")}
          </span>
          <ChevronDown
            size={16}
            aria-hidden
            style={{ transform: sheetExpanded ? "rotate(180deg)" : undefined }}
          />
        </button>
        <div id="map-side-body" className="map-side-body">
          {!selected && (
            <>
              <EmptyState title={t("map.selectLocation")} />
              {missionId && (
                <div style={{ padding: 16 }}>
                  <GuidancePanel missionId={missionId} screen="map" />
                </div>
              )}
            </>
          )}
          {selected && missionId && (
            <SelectedMarkerPanel missionId={missionId} marker={selected} />
          )}
        </div>
      </aside>
    </div>
  );
}

/**
 * Fallback tactical map board: positions Google-Maps-compatible markers on a
 * dark grid surface using normalized lat/lng.
 */
function FallbackBoard({
  markers,
  selectedId,
  onSelect,
}: {
  markers: Marker[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const positioned = useMemo(() => {
    if (markers.length === 0) return [];
    const lats = markers.map((m) => m.lat);
    const lngs = markers.map((m) => m.lng);
    const minLat = Math.min(...lats);
    const maxLat = Math.max(...lats);
    const minLng = Math.min(...lngs);
    const maxLng = Math.max(...lngs);
    const latSpan = maxLat - minLat || 1;
    const lngSpan = maxLng - minLng || 1;
    // 12% padding keeps labels inside the board.
    return markers.map((m) => ({
      marker: m,
      x: 12 + ((m.lng - minLng) / lngSpan) * 76,
      y: 12 + ((maxLat - m.lat) / latSpan) * 76,
    }));
  }, [markers]);

  return (
    <>
      {positioned.map(({ marker, x, y }) => (
        <button
          key={marker.id}
          className={[
            "marker-pin",
            marker.is_locked ? "locked" : "",
            marker.status === "visited" ? "visited" : "",
            selectedId === marker.id ? "selected" : "",
            marker.recommended && !marker.is_locked ? "recommended pulse" : "",
            marker.risk_level >= 60 || marker.objective_status === "high_risk"
              ? "highrisk"
              : "",
            marker.has_new_clue && !marker.is_locked ? "pulse" : "",
          ]
            .filter(Boolean)
            .join(" ")}
          style={{ left: `${x}%`, top: `${y}%` }}
          onClick={() => onSelect(marker.id)}
          aria-label={marker.name}
          aria-pressed={selectedId === marker.id}
        >
          <span className="marker-dot">
            {marker.is_locked ? (
              <Lock size={13} aria-hidden />
            ) : marker.status === "visited" ? (
              <CheckCircle2 size={13} aria-hidden />
            ) : (
              <MapPin size={13} aria-hidden />
            )}
            {marker.recommended && <span className="marker-flag recommended" />}
            {marker.has_new_clue && <span className="marker-flag clue" />}
            {marker.has_character && <span className="marker-flag character" />}
          </span>
          <span className="marker-label">{marker.name}</span>
        </button>
      ))}
    </>
  );
}

function SelectedMarkerPanel({
  missionId,
  marker,
}: {
  missionId: string;
  marker: Marker;
}) {
  const { t } = useI18n();
  const detail = useQuery({
    queryKey: ["mission", missionId, "location", marker.id],
    queryFn: () => mapApi.location(missionId, marker.id),
    enabled: !marker.is_locked,
  });

  const firstCharacter = detail.data?.characters[0];

  return (
    <div>
      <div className="band">
        <div className="spread">
          <h2>{marker.name}</h2>
          <span className="chip">{marker.type}</span>
        </div>
        <div className="row" style={{ marginTop: 8, flexWrap: "wrap" }}>
          {marker.is_locked ? (
            <span className="chip">{t("map.locked")}</span>
          ) : (
            <span className="chip chip-mission">
              {marker.status === "visited" ? t("map.visited") : t("map.discovered")}
            </span>
          )}
          <span
            className={`chip ${marker.risk_level >= 60 ? "chip-danger" : ""}`}
            title={t("map.riskLevel")}
          >
            <ShieldAlert size={12} aria-hidden />
            {t("map.riskLevel")}: {marker.risk_level}
          </span>
          {marker.recommended && (
            <span className="status-chip cat-guide">
              {t("map.marker.recommended")}
            </span>
          )}
          {marker.has_new_clue && (
            <span className="status-chip cat-neutral">{t("map.marker.newClue")}</span>
          )}
          {marker.has_character && (
            <span className="status-chip">{t("map.marker.character")}</span>
          )}
        </div>
      </div>

      {marker.is_locked && (
        <div className="band">
          <p className="muted">{t("map.lockedHint")}</p>
        </div>
      )}

      {!marker.is_locked && (
        <>
          {detail.isPending && <SkeletonRows rows={4} />}
          {detail.isError && (
            <ErrorState error={detail.error} onRetry={() => detail.refetch()} />
          )}
          {detail.data && (
            <div className="band">
              <p className="muted" style={{ unicodeBidi: "plaintext" }}>
                {detail.data.location.description}
              </p>
              {detail.data.characters.length > 0 && (
                <div style={{ marginTop: 12 }}>
                  <div className="band-title">{t("map.charactersHere")}</div>
                  <div className="row" style={{ flexWrap: "wrap" }}>
                    {detail.data.characters.map((c) => (
                      <Link
                        key={c.id}
                        to={`/app/missions/${missionId}/characters/${c.id}`}
                        className="chip"
                      >
                        <Avatar
                          name={c.name}
                          category={c.category}
                          imageUrl={c.avatar_url || undefined}
                          version={c.avatar_version}
                          size="sm"
                        />
                        {c.name}
                      </Link>
                    ))}
                  </div>
                </div>
              )}
              {detail.data.discovered_clues.length > 0 && (
                <div style={{ marginTop: 12 }}>
                  <div className="band-title">{t("map.cluesHere")}</div>
                  <div className="row" style={{ flexWrap: "wrap" }}>
                    {detail.data.discovered_clues.map((c) => (
                      <Link
                        key={c.id}
                        to={`/app/missions/${missionId}/clues/${c.id}`}
                        className="chip"
                      >
                        <Search size={12} aria-hidden />
                        {c.title}
                      </Link>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
          {/* Location actions: inspect / talk / ask AI / view clues */}
          <div className="band" style={{ borderBottom: "none" }}>
            <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
              <GameButton
                to={`/app/missions/${missionId}/locations/${marker.id}`}
                variant="primary"
              >
                <ExternalLink size={14} aria-hidden />
                {t("map.inspectArea")}
              </GameButton>
              {firstCharacter && (
                <GameButton
                  to={`/app/missions/${missionId}/characters/${firstCharacter.id}`}
                  variant="ghost"
                >
                  <MessageCircle size={14} aria-hidden />
                  {t("map.talk")}
                </GameButton>
              )}
              <GameButton
                to={`/app/missions/${missionId}/locations/${marker.id}?ask=1`}
                variant="ghost"
              >
                <Sparkles size={14} aria-hidden />
                {t("map.askAi")}
              </GameButton>
              <GameButton to={`/app/missions/${missionId}/clues`} variant="ghost">
                <Search size={14} aria-hidden />
                {t("map.viewClues")}
              </GameButton>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
