import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  MapPin,
  Lock,
  CheckCircle2,
  ExternalLink,
  ShieldAlert,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { mapApi } from "@/shared/api/endpoints";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import type { Marker } from "@/shared/types/api";
import { AvatarPlaceholder } from "@/shared/ui/badges";

/**
 * Fallback tactical map board: positions Google-Maps-compatible markers on a
 * dark grid surface using normalized lat/lng. Swappable for a real Maps
 * implementation without changing the data contract.
 */
export function MapPage() {
  const { missionId } = useParams<{ missionId: string }>();
  const { t } = useI18n();
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const map = useQuery({
    queryKey: ["mission", missionId, "map"],
    queryFn: () => mapApi.view(missionId!),
    enabled: !!missionId,
  });

  const positioned = useMemo(() => {
    const markers = map.data?.locations ?? [];
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
  }, [map.data]);

  const selected =
    map.data?.locations.find((m) => m.id === selectedId) ?? null;

  return (
    <div className="map-layout">
      <div className="map-board-wrap" role="application" aria-label={t("map.title")}>
        {map.isPending && <SkeletonRows rows={5} />}
        {map.isError && (
          <ErrorState error={map.error} onRetry={() => map.refetch()} />
        )}
        {positioned.map(({ marker, x, y }) => (
          <button
            key={marker.id}
            className={[
              "marker-pin",
              marker.is_locked ? "locked" : "",
              marker.status === "visited" ? "visited" : "",
              selectedId === marker.id ? "selected" : "",
              marker.has_new_clue && !marker.is_locked ? "pulse" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            style={{ left: `${x}%`, top: `${y}%` }}
            onClick={() => setSelectedId(marker.id)}
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
              {marker.has_new_clue && <span className="marker-flag clue" />}
              {marker.has_character && <span className="marker-flag character" />}
            </span>
            <span className="marker-label">{marker.name}</span>
          </button>
        ))}
      </div>

      <aside className="map-side" aria-label={t("map.selectLocation")}>
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
      </aside>
    </div>
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
            className={`chip ${marker.risk_level >= 3 ? "chip-danger" : ""}`}
            title={t("map.riskLevel")}
          >
            <ShieldAlert size={12} aria-hidden />
            {t("map.riskLevel")}: {marker.risk_level}
          </span>
          {marker.badge && <span className="chip chip-rare">{marker.badge}</span>}
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
                        <AvatarPlaceholder name={c.name} prompt={c.avatar_prompt} />
                        {c.name}
                      </Link>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
          <div className="band" style={{ borderBottom: "none" }}>
            <Link
              className="btn btn-primary"
              to={`/app/missions/${missionId}/locations/${marker.id}`}
            >
              <ExternalLink size={14} aria-hidden />
              {t("map.openLocation")}
            </Link>
          </div>
        </>
      )}
    </div>
  );
}
