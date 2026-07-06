import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Rocket, AlertTriangle } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { missionsApi, walletApi } from "@/shared/api/endpoints";
import { errorKey } from "@/shared/api/client";
import { CostBadge } from "@/shared/ui/badges";
import type {
  Difficulty,
  Language,
  MissionType,
} from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

const MISSION_TYPES: MissionType[] = [
  "detective",
  "wildlife_rescue",
  "disaster_response",
  "exploration",
  "survival",
  "diplomacy",
  "medical_mystery",
];

const DIFFICULTIES: Difficulty[] = ["easy", "medium", "hard", "expert"];

export function NewMissionPage() {
  const { t, lang } = useI18n();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const presetType = searchParams.get("type");
  const [type, setType] = useState<MissionType>(
    MISSION_TYPES.includes(presetType as MissionType)
      ? (presetType as MissionType)
      : "detective",
  );
  const [difficulty, setDifficulty] = useState<Difficulty>("medium");
  const [region, setRegion] = useState("");
  const [language, setLanguage] = useState<Language>(lang);

  const wallet = useQuery({ queryKey: ["wallet"], queryFn: walletApi.get });
  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
  });

  const creationCost = pricing.data?.mission_start ?? null;
  const insufficient =
    creationCost !== null &&
    wallet.data !== undefined &&
    wallet.data.balance < creationCost;

  const create = useMutation({
    mutationFn: () =>
      missionsApi.create({
        type,
        difficulty,
        region: region.trim() || undefined,
        language,
      }),
    onSuccess: (mission) => {
      void queryClient.invalidateQueries({ queryKey: ["missions"] });
      void queryClient.invalidateQueries({ queryKey: ["wallet"] });
      navigate(`/app/missions/${mission.id}`);
    },
  });

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!create.isPending) create.mutate();
  };

  return (
    <div className="page" style={{ maxWidth: 720 }}>
      <header className="page-header">
        <h1>{t("missions.new.title")}</h1>
        {creationCost !== null && (
          <div className="row">
            <span className="faint">{t("missions.new.cost")}</span>
            <CostBadge coins={creationCost} />
          </div>
        )}
      </header>

      <form className="stack" style={{ gap: 20 }} onSubmit={onSubmit}>
        <fieldset className="field" style={{ border: "none", padding: 0, margin: 0 }}>
          <legend className="field-label" style={{ marginBottom: 6 }}>
            {t("missions.new.type")}
          </legend>
          <div className="row" style={{ flexWrap: "wrap", gap: 6 }}>
            {MISSION_TYPES.map((mt) => (
              <button
                key={mt}
                type="button"
                className={`btn ${type === mt ? "btn-primary" : "btn-secondary"}`}
                aria-pressed={type === mt}
                onClick={() => setType(mt)}
              >
                {t(`type.${mt}` as TranslationKey)}
              </button>
            ))}
          </div>
        </fieldset>

        <fieldset className="field" style={{ border: "none", padding: 0, margin: 0 }}>
          <legend className="field-label" style={{ marginBottom: 6 }}>
            {t("missions.new.difficulty")}
          </legend>
          <div className="segmented">
            {DIFFICULTIES.map((d) => (
              <button
                key={d}
                type="button"
                aria-pressed={difficulty === d}
                onClick={() => setDifficulty(d)}
              >
                {t(`difficulty.${d}` as TranslationKey)}
              </button>
            ))}
          </div>
        </fieldset>

        <div className="field" style={{ maxWidth: 340 }}>
          <label className="field-label" htmlFor="region">
            {t("missions.new.region")}
          </label>
          <input
            id="region"
            className="input"
            value={region}
            placeholder={t("missions.new.region.placeholder")}
            onChange={(e) => setRegion(e.target.value)}
          />
        </div>

        <div className="field">
          <span className="field-label">{t("missions.new.language")}</span>
          <div className="segmented">
            <button
              type="button"
              aria-pressed={language === "en"}
              onClick={() => setLanguage("en")}
            >
              English
            </button>
            <button
              type="button"
              aria-pressed={language === "fa"}
              onClick={() => setLanguage("fa")}
            >
              فارسی
            </button>
          </div>
        </div>

        {insufficient && (
          <p className="field-error row" role="alert">
            <AlertTriangle size={14} aria-hidden />
            {t("missions.new.insufficient")}
          </p>
        )}
        {create.isError && (
          <p className="field-error" role="alert">
            {t(errorKey(create.error))}
          </p>
        )}

        <div>
          <button
            className="btn btn-primary"
            type="submit"
            disabled={create.isPending || insufficient}
          >
            <Rocket size={15} aria-hidden />
            {create.isPending ? t("common.loading") : t("missions.new.cta")}
          </button>
        </div>
      </form>
    </div>
  );
}
