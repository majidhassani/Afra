import { useEffect, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Coins,
  PlayCircle,
  ShieldCheck,
  Gift,
  Sparkles,
  Info,
  FlaskConical,
} from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { walletApi } from "@/shared/api/endpoints";
import { ApiError, errorKey } from "@/shared/api/client";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { WalletBalance } from "@/shared/ui/game";
import { toast } from "@/shared/ui/toast";
import { Button } from "@/shared/ui/Button";
import { env } from "@/shared/config/env";
import type { CoinPack } from "@/shared/types/api";
import type { TranslationKey } from "@/shared/i18n/en";

function actionLabel(
  t: (k: TranslationKey) => string,
  action: string,
): string {
  const key = `wallet.action.${action}` as TranslationKey;
  const label = t(key);
  return label === key ? action.replace(/_/g, " ") : label;
}

const packTier: Record<string, string> = {
  coins_small: "common",
  coins_medium: "rare",
  coins_large: "epic",
};

export function WalletPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  // Coin packs charge real money (or a simulated receipt in demo mode) — the
  // store used to fire the purchase on the very first tap. This now requires
  // a second tap on the same pack within a few seconds, mirroring the
  // "tap again to confirm" pattern used for time-costing actions elsewhere
  // in the app, so a purchase can't happen from a single stray tap.
  const [pendingPackId, setPendingPackId] = useState<string | null>(null);
  const pendingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => () => {
    if (pendingTimer.current) clearTimeout(pendingTimer.current);
  }, []);
  const armPack = (id: string) => {
    setPendingPackId(id);
    if (pendingTimer.current) clearTimeout(pendingTimer.current);
    pendingTimer.current = setTimeout(() => setPendingPackId(null), 4000);
  };

  const wallet = useQuery({ queryKey: ["wallet"], queryFn: walletApi.get });
  const config = useQuery({
    queryKey: ["wallet", "config"],
    queryFn: walletApi.config,
  });
  const pricing = useQuery({
    queryKey: ["wallet", "pricing"],
    queryFn: walletApi.pricing,
  });
  const transactions = useQuery({
    queryKey: ["wallet", "transactions", 50],
    queryFn: () => walletApi.transactions(50),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["wallet"] });
  };

  const claimAd = useMutation({
    mutationFn: walletApi.claimRewardedAd,
    onSuccess: () => {
      toast("success", t("wallet.adClaimed"));
      invalidate();
    },
    onError: (err) => {
      if (err instanceof ApiError && err.status === 409) {
        toast("error", t("wallet.adLimit"));
      } else {
        toast("error", t(errorKey(err)));
      }
    },
  });

  const buyPack = useMutation({
    mutationFn: (pack: CoinPack) =>
      walletApi.verifyPurchase({
        platform: "android",
        product_id: pack.product_id,
        // Demo receipt — real store verification is server-gated.
        receipt: `demo-${pack.product_id}-${Date.now()}`,
      }),
    onSuccess: () => {
      toast("success", t("wallet.purchaseVerified"));
      invalidate();
    },
    onError: (err) => toast("error", t(errorKey(err))),
    onSettled: () => setPendingPackId(null),
  });

  const demoEnabled = config.data?.demo_purchases ?? false;
  const coinPacks = config.data?.coin_packs ?? [];

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("wallet.title")}</h1>
        {demoEnabled && (
          <span className="chip chip-rare" title={t("wallet.demo.body")}>
            <FlaskConical size={12} aria-hidden />
            {t("wallet.demo")}
          </span>
        )}
      </header>

      {/* Balance HUD */}
      <div className="wallet-hud">
        <div className="wallet-balance-card">
          <span className="eyebrow row" style={{ gap: 6 }}>
            <Coins size={13} aria-hidden />
            {t("wallet.balance")}
          </span>
          <WalletBalance balance={wallet.data?.balance} />
          {wallet.data && wallet.data.reserved_balance > 0 && (
            <span className="faint" style={{ fontSize: 12 }}>
              {t("wallet.reserved")}: {wallet.data.reserved_balance}
            </span>
          )}
        </div>
        <button
          className="wallet-reward-card"
          disabled={claimAd.isPending || !demoEnabled}
          onClick={() => claimAd.mutate()}
        >
          <span className="wr-icon" aria-hidden>
            <Gift size={20} />
          </span>
          <span className="grow" style={{ textAlign: "start" }}>
            <strong>{t("wallet.dailyReward")}</strong>
            <span className="faint" style={{ display: "block", fontSize: 12 }}>
              {t("wallet.dailyReward.body", {
                coins: String(config.data?.rewarded_ad_coins ?? 20),
              })}
            </span>
          </span>
          <PlayCircle size={18} aria-hidden />
        </button>
      </div>

      {/* Coin packs (game economy store) */}
      <section aria-label={t("wallet.store")} style={{ marginTop: 18 }}>
        <div className="band-title">{t("wallet.store")}</div>
        {config.isPending && (
          <div className="panel">
            <SkeletonRows rows={2} />
          </div>
        )}
        <div className="coin-pack-grid">
          {coinPacks.map((pack) => {
            const confirming = pendingPackId === pack.product_id;
            return (
              <button
                key={pack.product_id}
                className={`coin-pack tier-${packTier[pack.product_id] ?? "common"}${confirming ? " confirm" : ""}`}
                disabled={buyPack.isPending || !demoEnabled}
                aria-label={
                  demoEnabled
                    ? `${t("wallet.getPack")}: ${pack.coins} ${t("wallet.coins")}${confirming ? ` — ${t("wallet.confirmPurchase")}` : ""}`
                    : t("wallet.storeSoon")
                }
                onClick={() => {
                  if (confirming) {
                    if (pendingTimer.current) clearTimeout(pendingTimer.current);
                    setPendingPackId(null);
                    buyPack.mutate(pack);
                  } else {
                    armPack(pack.product_id);
                  }
                }}
              >
                <span className="cp-shine" aria-hidden />
                <Coins size={26} className="cp-coin" aria-hidden />
                <span className="cp-amount mono-num">{pack.coins}</span>
                <span className="cp-label">{t("wallet.coins")}</span>
                <span className="cp-cta">
                  {confirming ? (
                    <ShieldCheck size={12} aria-hidden />
                  ) : (
                    <Sparkles size={12} aria-hidden />
                  )}
                  {!demoEnabled
                    ? t("wallet.storeSoon")
                    : confirming
                      ? t("wallet.confirmPurchase")
                      : t("wallet.getPack")}
                </span>
              </button>
            );
          })}
        </div>
      </section>

      <div className="dash-grid" style={{ marginTop: 18 }}>
        {/* AI cost explanation */}
        <section className="panel col-4" aria-label={t("wallet.pricing")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("wallet.pricing")}
          </div>
          <p className="faint row" style={{ padding: "0 16px 8px", gap: 6 }}>
            <Info size={12} aria-hidden />
            {t("wallet.pricing.body")}
          </p>
          {pricing.isPending && <SkeletonRows rows={4} />}
          {pricing.isError && (
            <ErrorState error={pricing.error} onRetry={() => pricing.refetch()} />
          )}
          <div className="item-list">
            {pricing.data &&
              Object.entries(pricing.data).map(([action, price]) => (
                <div key={action} className="item-row">
                  <span className="grow sub">{actionLabel(t, action)}</span>
                  <span className="chip chip-wallet mono-num">
                    <Coins size={11} aria-hidden />
                    {price}
                  </span>
                </div>
              ))}
          </div>
        </section>

        {/* Transaction ledger */}
        <section className="panel col-8" aria-label={t("wallet.transactions")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("wallet.transactions")}
          </div>
          {transactions.isPending && <SkeletonRows rows={6} />}
          {transactions.isError && (
            <ErrorState
              error={transactions.error}
              onRetry={() => transactions.refetch()}
            />
          )}
          {transactions.isSuccess && transactions.data.length === 0 && (
            <EmptyState
              title={t("wallet.transactions.empty")}
              body={t("wallet.transactions.empty.body")}
            />
          )}
          {transactions.isSuccess && transactions.data.length > 0 && (
            <div className="item-list">
              {transactions.data.map((txn) => (
                <div key={txn.id} className="item-row">
                  <span
                    className={`txn-dir ${txn.amount >= 0 ? "in" : "out"}`}
                    aria-hidden
                  >
                    <Coins size={13} />
                  </span>
                  <span className="grow">
                    <span className="title">{actionLabel(t, txn.type)}</span>
                    <span className="sub mono-num faint">
                      {new Date(txn.created_at).toLocaleString()}
                    </span>
                  </span>
                  <span
                    className="mono-num"
                    style={{
                      fontWeight: 600,
                      color:
                        txn.amount >= 0
                          ? "var(--accent-mission)"
                          : "var(--accent-danger)",
                    }}
                  >
                    {txn.amount >= 0 ? `+${txn.amount}` : txn.amount}
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      {/* Developer-only receipt verification tool (never in production UI). */}
      {env.enableDiagnostics && demoEnabled && <ReceiptTestForm onDone={invalidate} />}
    </div>
  );
}

/** Dev-only manual receipt verification form (hidden from the game UI). */
function ReceiptTestForm({ onDone }: { onDone: () => void }) {
  const { t } = useI18n();
  const [platform, setPlatform] = useState<"ios" | "android">("android");
  const [productId, setProductId] = useState("coins_small");
  const [receipt, setReceipt] = useState("");

  const verify = useMutation({
    mutationFn: () =>
      walletApi.verifyPurchase({ platform, product_id: productId, receipt }),
    onSuccess: () => {
      toast("success", t("wallet.purchaseVerified"));
      setReceipt("");
      onDone();
    },
    onError: (err) => toast("error", t(errorKey(err))),
  });

  const onVerify = (e: FormEvent) => {
    e.preventDefault();
    if (receipt.trim() && !verify.isPending) verify.mutate();
  };

  return (
    <section
      className="panel"
      style={{ marginTop: 18, borderColor: "var(--accent-rare-dim)" }}
      aria-label={t("wallet.purchase")}
    >
      <div className="band-title row" style={{ padding: "14px 16px 0", gap: 6 }}>
        <FlaskConical size={13} aria-hidden />
        {t("wallet.devReceipt")}
      </div>
      <form
        className="row"
        style={{ padding: 16, flexWrap: "wrap", alignItems: "flex-end" }}
        onSubmit={onVerify}
      >
        <div className="field">
          <span className="field-label">{t("wallet.platform")}</span>
          <div className="segmented">
            <button
              type="button"
              aria-pressed={platform === "android"}
              onClick={() => setPlatform("android")}
            >
              Android
            </button>
            <button
              type="button"
              aria-pressed={platform === "ios"}
              onClick={() => setPlatform("ios")}
            >
              iOS
            </button>
          </div>
        </div>
        <div className="field">
          <label className="field-label" htmlFor="product">
            {t("wallet.product")}
          </label>
          <select
            id="product"
            className="select"
            value={productId}
            onChange={(e) => setProductId(e.target.value)}
          >
            <option value="coins_small">coins_small</option>
            <option value="coins_medium">coins_medium</option>
            <option value="coins_large">coins_large</option>
          </select>
        </div>
        <div className="field" style={{ flex: 1, minWidth: 200 }}>
          <label className="field-label" htmlFor="receipt">
            {t("wallet.receipt")}
          </label>
          <input
            id="receipt"
            className="input"
            value={receipt}
            onChange={(e) => setReceipt(e.target.value)}
          />
        </div>
        <Button
          variant="secondary"
          type="submit"
          loading={verify.isPending}
          disabled={!receipt.trim()}
        >
          <ShieldCheck size={14} aria-hidden />
          {t("wallet.purchase")}
        </Button>
      </form>
    </section>
  );
}
