import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  ShieldAlert,
  Search,
  ScanLine,
  FileText,
  Eye,
  Play,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { useLanguageGuard } from "@/shared/i18n/languageGuard";
import { mapApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows } from "@/shared/ui/states";
import { CostBadge } from "@/shared/ui/badges";
import { Analyzing } from "@/shared/ui/game";
import { Avatar } from "@/shared/ui/Avatar";
import { Button } from "@/shared/ui/Button";
import { GuidancePanel } from "@/features/guidance/GuidancePanel";
import { ActionTimePreview } from "@/features/game/ActionTimePreview";
import { announceTimeUpdate } from "@/features/game/gameEvents";
import type { ActionResult } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

const actionIcons: Record<string, typeof Search> = {
  inspect_area: Search,
  scan_environment: ScanLine,
  review_documents: FileText,
  observe: Eye,
  search: Search,
};

function actionLabel(
  t: (k: TranslationKey, p?: Record<string, string | number>) => string,
  action: string,
): string {
  const key = `map.action.${action}` as TranslationKey;
  const label = t(key);
  return label === key ? action.replace(/_/g, " ") : label;
}

export function LocationPage() {
  const { missionId, locationId } = useParams<{
    missionId: string;
    locationId: string;
  }>();
  const { t } = useI18n();
  const guardLanguage = useLanguageGuard();
  const queryClient = useQueryClient();
  const [lastResult, setLastResult] = useState<ActionResult | null>(null);
  const [pendingAction, setPendingAction] = useState<string | null>(null);

  const detail = useQuery({
    queryKey: ["mission", missionId, "location", locationId],
    queryFn: () => mapApi.location(missionId!, locationId!),
    enabled: !!missionId && !!locationId,
  });

  // First visits cost travel time — surface it once when the detail loads.
  const visitTime = detail.data?.time_update;
  useEffect(() => {
    if (visitTime) announceTimeUpdate(visitTime);
  }, [visitTime]);

  const runAction = useMutation({
    mutationFn: (action: string) => mapApi.runAction(missionId!, locationId!, action),
    onSuccess: (result) => {
      guardLanguage(result.narrative);
      setLastResult(result);
      announceTimeUpdate(result.time_update);
      void queryClient.invalidateQueries({ queryKey: ["mission", missionId] });
      void queryClient.invalidateQueries({ queryKey: ["gameplay-status", missionId] });
      void queryClient.invalidateQueries({ queryKey: ["timeline", missionId] });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
    },
  });

  if (detail.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={7} />
      </div>
    );
  }
  if (detail.isError) {
    return (
      <div className="page">
        <ErrorState error={detail.error} onRetry={() => detail.refetch()} />
      </div>
    );
  }

  const { location, characters, discovered_clues } = detail.data;
  const actions = Array.isArray(location.available_actions)
    ? location.available_actions
    : [];

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <Link
            className="row faint"
            to={`/app/missions/${missionId}/map`}
            style={{ marginBottom: 6 }}
          >
            <ArrowLeft size={13} className="rtl-flip" aria-hidden />
            {t("map.title")}
          </Link>
          <h1>{location.name}</h1>
          <div className="row subtitle" style={{ flexWrap: "wrap" }}>
            <span className="chip">{location.type}</span>
            <span className={`chip ${location.risk_level >= 3 ? "chip-danger" : ""}`}>
              <ShieldAlert size={12} aria-hidden />
              {t("map.riskLevel")}: {location.risk_level}
            </span>
          </div>
        </div>
      </header>

      <div className="dash-grid">
        <section className="panel col-8">
          <div className="band">
            <p style={{ unicodeBidi: "plaintext" }}>{location.description}</p>
          </div>
          <div className="band">
            <div className="band-title">{t("map.actions")}</div>
            <div className="row" style={{ flexWrap: "wrap" }}>
              {actions.map((action) => {
                const Icon = actionIcons[action] ?? Play;
                return (
                  <Button
                    key={action}
                    variant="ghost"
                    size="sm"
                    disabled={runAction.isPending}
                    onClick={() => setPendingAction(action)}
                  >
                    <Icon size={14} aria-hidden />
                    {actionLabel(t, action)}
                  </Button>
                );
              })}
            </div>
            {/* Time-cost confirmation: every action shows its price first. */}
            {pendingAction && (
              <ActionTimePreview
                missionId={missionId!}
                action="location_action"
                targetId={locationId}
                onConfirm={() => {
                  runAction.mutate(pendingAction);
                  setPendingAction(null);
                }}
                onCancel={() => setPendingAction(null)}
              />
            )}
            {runAction.isPending && <Analyzing label={t("loading.analyzing")} />}
            {runAction.isError && (
              <p className="field-error" role="alert" style={{ marginTop: 12 }}>
                {t(errorKey(runAction.error))}
              </p>
            )}
            {lastResult && !runAction.isPending && (
              <div className="stack" style={{ marginTop: 12 }}>
                <div className="band-title">{t("map.narrative")}</div>
                <p style={{ unicodeBidi: "plaintext" }}>{lastResult.narrative}</p>
                {lastResult.new_facts.length > 0 && (
                  <>
                    <div className="band-title">{t("map.newFacts")}</div>
                    <ul style={{ margin: 0, paddingInlineStart: 18 }}>
                      {lastResult.new_facts.map((fact) => (
                        <li key={fact} className="muted">
                          {fact}
                        </li>
                      ))}
                    </ul>
                  </>
                )}
                {lastResult.discovered_clues.length > 0 && (
                  <>
                    <div className="band-title">{t("map.discoveredClues")}</div>
                    <div className="row" style={{ flexWrap: "wrap" }}>
                      {lastResult.discovered_clues.map((clue) => (
                        <Link
                          key={clue.id}
                          className="chip chip-wallet"
                          to={`/app/missions/${missionId}/clues/${clue.id}`}
                        >
                          {clue.title}
                        </Link>
                      ))}
                    </div>
                  </>
                )}
                <CostBadge coins={lastResult.cost.coins_charged} />
              </div>
            )}
          </div>
          <div className="band" style={{ borderBottom: "none" }}>
            <GuidancePanel
              missionId={missionId!}
              screen="location"
              locationId={locationId}
            />
          </div>
        </section>

        <div className="col-4 stack">
          <section className="panel" aria-label={t("map.charactersHere")}>
            <div className="band-title" style={{ padding: "14px 16px 0" }}>
              {t("map.charactersHere")}
            </div>
            <div className="item-list">
              {characters.map((c) => (
                <Link
                  key={c.id}
                  className="item-row"
                  to={`/app/missions/${missionId}/characters/${c.id}`}
                >
                  <Avatar name={c.name} category={c.category} size="sm" />
                  <div className="grow">
                    <div className="title">{c.name}</div>
                    <div className="sub">{c.role}</div>
                  </div>
                </Link>
              ))}
              {characters.length === 0 && (
                <div className="item-row faint">{t("common.empty.title")}</div>
              )}
            </div>
          </section>

          <section className="panel" aria-label={t("map.cluesHere")}>
            <div className="band-title" style={{ padding: "14px 16px 0" }}>
              {t("map.cluesHere")}
            </div>
            <div className="item-list">
              {discovered_clues.map((clue) => (
                <Link
                  key={clue.id}
                  className="item-row"
                  to={`/app/missions/${missionId}/clues/${clue.id}`}
                >
                  <Search size={14} aria-hidden />
                  <div className="grow">
                    <div className="title">{clue.title}</div>
                    <div className="sub">{clue.short_description}</div>
                  </div>
                </Link>
              ))}
              {discovered_clues.length === 0 && (
                <div className="item-row faint">{t("common.empty.title")}</div>
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
