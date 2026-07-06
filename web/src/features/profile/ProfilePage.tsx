import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BadgeCheck,
  Pencil,
  Sparkles,
  Rocket,
  Trophy,
  CircleX,
  Target,
  Search,
  MapPin,
  Coins,
  ShieldHalf,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { avatarApi, profileApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { ErrorState, SkeletonRows, EmptyState } from "@/shared/ui/states";
import { Avatar } from "@/shared/ui/Avatar";
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

  const successRate = Math.round(
    profile.success_rate <= 1 ? profile.success_rate * 100 : profile.success_rate,
  );
  // XP progress within the current rank (assume 500 XP per level as flavor;
  // the raw XP total is shown alongside so nothing is misrepresented).
  const xpIntoLevel = ((profile.xp % 500) / 500) * 100;

  const achievements: Array<{
    key: TranslationKey;
    value: string | number;
    icon: typeof Rocket;
    tone: string;
  }> = [
    { key: "profile.totalMissions", value: profile.total_missions, icon: Rocket, tone: "ai" },
    { key: "profile.completed", value: profile.completed_missions, icon: Trophy, tone: "mission" },
    { key: "profile.failed", value: profile.failed_missions, icon: CircleX, tone: "danger" },
    { key: "profile.successRate", value: `${successRate}%`, icon: Target, tone: "wallet" },
    { key: "profile.cluesFound", value: profile.total_clues_found, icon: Search, tone: "ai" },
    { key: "profile.aiInteractions", value: profile.total_ai_interactions, icon: Sparkles, tone: "rare" },
    { key: "profile.locationsVisited", value: profile.total_locations_visited, icon: MapPin, tone: "mission" },
    { key: "profile.coinsSpent", value: stats.data.total_coins_spent, icon: Coins, tone: "danger" },
    { key: "profile.coinsEarned", value: stats.data.total_coins_earned, icon: Coins, tone: "mission" },
  ];

  return (
    <div className="page">
      {/* Agent progression crest */}
      <section className="agent-crest">
        <div className="crest-avatar">
          <Avatar name={profile.display_name} category="guide" size="xl" glow />
          <span className="crest-level mono-num" aria-hidden>
            {profile.level}
          </span>
        </div>
        <div className="grow" style={{ minWidth: 0 }}>
          <span className="eyebrow row" style={{ gap: 6, color: "var(--accent-ai)" }}>
            <ShieldHalf size={13} aria-hidden />
            {profile.rank}
          </span>
          <h1 style={{ marginTop: 2 }}>{profile.display_name}</h1>
          <div className="crest-xp">
            <div className="crest-xp-head">
              <span>
                {t("profile.level")} {profile.level}
              </span>
              <span className="mono-num">
                {profile.xp} {t("profile.xp")}
              </span>
            </div>
            <div className="xp-bar" role="img" aria-label={`${profile.xp} XP`}>
              <span style={{ width: `${xpIntoLevel}%` }} />
            </div>
          </div>
          {profile.favorite_mission_type && (
            <span className="status-chip" style={{ marginTop: 10 }}>
              {t("profile.favoriteType")}:{" "}
              {t(`type.${profile.favorite_mission_type}` as TranslationKey)}
            </span>
          )}
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
      </section>

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

      <div className="band-title" style={{ margin: "18px 0 10px" }}>
        {t("profile.achievements")}
      </div>
      <div className="achv-grid">
        {achievements.map((a) => (
          <div key={a.key} className={`achv-tile tone-${a.tone}`}>
            <span className="achv-icon" aria-hidden>
              <a.icon size={18} />
            </span>
            <span className="achv-value mono-num">{a.value}</span>
            <span className="achv-label">{t(a.key)}</span>
          </div>
        ))}
      </div>

      <div className="dash-grid" style={{ marginTop: 14 }}>
        <AvatarGeneratorSection seed={profile.display_name} />

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

/** Avatar generator: pick style/gender/age, generate a transparent PNG. */
function AvatarGeneratorSection({ seed }: { seed: string }) {
  const { t } = useI18n();
  const [style, setStyle] = useState("modern-minimal");
  const [gender, setGender] = useState("unspecified");
  const [ageGroup, setAgeGroup] = useState("adult");

  const options = useQuery({
    queryKey: ["avatar", "options"],
    queryFn: avatarApi.options,
  });

  const generate = useMutation({
    mutationFn: () =>
      avatarApi.generate({ style, gender, age_group: ageGroup, seed }),
  });

  const avatar = generate.data;

  return (
    <section className="panel col-12" aria-label={t("avatar.title")}>
      <div className="band-title" style={{ padding: "14px 16px 0" }}>
        {t("avatar.title")}
      </div>
      <div className="row" style={{ flexWrap: "wrap", padding: 16, gap: 12 }}>
        {avatar && (
          <img
            src={`data:${avatar.mime};base64,${avatar.png_base64}`}
            alt={t("avatar.title")}
            style={{ width: 96, height: 96, borderRadius: 12 }}
          />
        )}
        <div className="field">
          <label className="field-label" htmlFor="avatarStyle">
            {t("avatar.style")}
          </label>
          <select
            id="avatarStyle"
            className="input"
            value={style}
            onChange={(e) => setStyle(e.target.value)}
          >
            {(options.data?.styles ?? [style]).map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field-label" htmlFor="avatarGender">
            {t("avatar.gender")}
          </label>
          <select
            id="avatarGender"
            className="input"
            value={gender}
            onChange={(e) => setGender(e.target.value)}
          >
            {(options.data?.genders ?? [gender]).map((g) => (
              <option key={g} value={g}>
                {g}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label className="field-label" htmlFor="avatarAge">
            {t("avatar.ageGroup")}
          </label>
          <select
            id="avatarAge"
            className="input"
            value={ageGroup}
            onChange={(e) => setAgeGroup(e.target.value)}
          >
            {(options.data?.age_groups ?? [ageGroup]).map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </div>
        <button
          className="btn btn-primary"
          type="button"
          disabled={generate.isPending}
          onClick={() => generate.mutate()}
        >
          <Sparkles size={14} aria-hidden />
          {generate.isPending ? t("avatar.generating") : t("avatar.generate")}
        </button>
        {generate.isError && (
          <p className="field-error" role="alert">
            {t(errorKey(generate.error))}
          </p>
        )}
      </div>
    </section>
  );
}
