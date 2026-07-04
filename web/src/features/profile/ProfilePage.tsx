import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { BadgeCheck, Pencil } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { profileApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import { toast } from "@/shared/ui/toast";
import type { TranslationKey } from "@/shared/i18n/en";

interface Badge {
  id?: string;
  name?: string;
}

export function ProfilePage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");

  const stats = useQuery({
    queryKey: ["profile", "stats"],
    queryFn: profileApi.stats,
  });

  const save = useMutation({
    mutationFn: () => profileApi.update(name.trim()),
    onSuccess: () => {
      toast("success", t("profile.saved"));
      setEditing(false);
      void queryClient.invalidateQueries({ queryKey: ["profile"] });
      void queryClient.invalidateQueries({ queryKey: ["profile", "stats"] });
    },
  });

  if (stats.isPending) {
    return (
      <div className="page">
        <SkeletonRows rows={8} />
      </div>
    );
  }
  if (stats.isError) {
    return (
      <div className="page">
        <ErrorState error={stats.error} onRetry={() => stats.refetch()} />
      </div>
    );
  }

  const profile = stats.data.profile;
  const badges: Badge[] = Array.isArray(profile.badges)
    ? (profile.badges as Badge[])
    : [];

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (name.trim() && !save.isPending) save.mutate();
  };

  const numbers: Array<[TranslationKey, string | number]> = [
    ["profile.totalMissions", profile.total_missions],
    ["profile.completed", profile.completed_missions],
    ["profile.failed", profile.failed_missions],
    [
      "profile.successRate",
      `${Math.round((profile.success_rate <= 1 ? profile.success_rate * 100 : profile.success_rate))}%`,
    ],
    ["profile.cluesFound", profile.total_clues_found],
    ["profile.aiInteractions", profile.total_ai_interactions],
    ["profile.locationsVisited", profile.total_locations_visited],
    ["profile.coinsSpent", stats.data.total_coins_spent],
    ["profile.coinsEarned", stats.data.total_coins_earned],
  ];

  return (
    <div className="page">
      <header className="page-header">
        <div>
          <h1>{profile.display_name}</h1>
          <div className="row subtitle" style={{ flexWrap: "wrap" }}>
            <span className="chip chip-ai">{profile.rank}</span>
            <span className="chip mono-num">
              {t("profile.level")} {profile.level}
            </span>
            <span className="chip mono-num">
              {profile.xp} {t("profile.xp")}
            </span>
            {profile.favorite_mission_type && (
              <span className="chip">
                {t("profile.favoriteType")}:{" "}
                {t(`type.${profile.favorite_mission_type}` as TranslationKey)}
              </span>
            )}
          </div>
        </div>
        {!editing && (
          <button
            className="btn btn-secondary"
            onClick={() => {
              setName(profile.display_name);
              setEditing(true);
            }}
          >
            <Pencil size={14} aria-hidden />
            {t("common.edit")}
          </button>
        )}
      </header>

      {editing && (
        <form
          className="panel row"
          style={{ padding: 16, marginBottom: 14, flexWrap: "wrap" }}
          onSubmit={onSubmit}
        >
          <div className="field" style={{ flex: 1, minWidth: 220 }}>
            <label className="field-label" htmlFor="displayName">
              {t("profile.displayName")}
            </label>
            <input
              id="displayName"
              className="input"
              value={name}
              maxLength={60}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <button
            className="btn btn-primary"
            type="submit"
            disabled={save.isPending || !name.trim()}
          >
            {t("common.save")}
          </button>
          <button
            className="btn btn-ghost"
            type="button"
            onClick={() => setEditing(false)}
          >
            {t("common.cancel")}
          </button>
          {save.isError && (
            <p className="field-error" role="alert">
              {t(errorKey(save.error))}
            </p>
          )}
        </form>
      )}

      <div className="dash-grid">
        {numbers.map(([key, value]) => (
          <div key={key} className="panel stat-block col-4 col-half-sm">
            <span className="label">{t(key)}</span>
            <span className="value">{value}</span>
          </div>
        ))}

        <section className="panel col-12" aria-label={t("profile.badges")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("profile.badges")}
          </div>
          {badges.length === 0 ? (
            <EmptyState title={t("profile.badges.empty")} />
          ) : (
            <div className="row" style={{ flexWrap: "wrap", padding: 16 }}>
              {badges.map((badge, i) => (
                <span key={badge.id ?? i} className="chip chip-rare">
                  <BadgeCheck size={13} aria-hidden />
                  {badge.name ?? badge.id}
                </span>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
