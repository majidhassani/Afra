import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Coins, PlayCircle, ShieldCheck } from "lucide-react";
import { useI18n } from "@/shared/i18n";
import { walletApi } from "@/shared/api/endpoints";
import { ApiError, errorKey } from "@/shared/api/client";
import { EmptyState, ErrorState, SkeletonRows } from "@/shared/ui/states";
import { WalletBalance } from "@/shared/ui/game";
import { toast } from "@/shared/ui/toast";
import type { TranslationKey } from "@/shared/i18n/en";

function actionLabel(
  t: (k: TranslationKey) => string,
  action: string,
): string {
  const key = `wallet.action.${action}` as TranslationKey;
  const label = t(key);
  return label === key ? action.replace(/_/g, " ") : label;
}

export function WalletPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const wallet = useQuery({ queryKey: ["wallet"], queryFn: walletApi.get });
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

  const [platform, setPlatform] = useState<"ios" | "android">("android");
  const [productId, setProductId] = useState("coins_small");
  const [receipt, setReceipt] = useState("");

  const verify = useMutation({
    mutationFn: () =>
      walletApi.verifyPurchase({ platform, product_id: productId, receipt }),
    onSuccess: () => {
      toast("success", t("wallet.purchaseVerified"));
      setReceipt("");
      invalidate();
    },
    onError: (err) => toast("error", t(errorKey(err))),
  });

  const onVerify = (e: FormEvent) => {
    e.preventDefault();
    if (receipt.trim() && !verify.isPending) verify.mutate();
  };

  return (
    <div className="page">
      <header className="page-header">
        <h1>{t("wallet.title")}</h1>
      </header>

      <div className="dash-grid">
        <div className="tac-card col-4 col-half-sm stat-block" style={{ gap: 12 }}>
          <span className="label">{t("wallet.balance")}</span>
          <WalletBalance balance={wallet.data?.balance} />
        </div>
        <div className="panel stat-block col-4 col-half-sm">
          <span className="label">{t("wallet.reserved")}</span>
          <span className="value">
            {wallet.data ? wallet.data.reserved_balance : "—"}
          </span>
        </div>
        <div className="panel stat-block col-4">
          <span className="label">{t("wallet.claimAd")}</span>
          <div>
            <button
              className="btn btn-secondary"
              disabled={claimAd.isPending}
              onClick={() => claimAd.mutate()}
            >
              <PlayCircle size={14} aria-hidden />
              {t("wallet.claimAd")}
            </button>
          </div>
        </div>

        <section className="panel col-4" aria-label={t("wallet.pricing")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("wallet.pricing")}
          </div>
          <p className="faint" style={{ padding: "0 16px 8px" }}>
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
            <EmptyState title={t("common.empty.title")} />
          )}
          {transactions.isSuccess && transactions.data.length > 0 && (
            <div style={{ overflowX: "auto" }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t("wallet.txn.type")}</th>
                    <th>{t("wallet.txn.amount")}</th>
                    <th>{t("wallet.txn.after")}</th>
                    <th>{t("wallet.txn.date")}</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.data.map((txn) => (
                    <tr key={txn.id}>
                      <td>{actionLabel(t, txn.type)}</td>
                      <td
                        className="mono-num"
                        style={{
                          color:
                            txn.amount >= 0
                              ? "var(--accent-mission)"
                              : "var(--accent-danger)",
                        }}
                      >
                        {txn.amount >= 0 ? `+${txn.amount}` : txn.amount}
                      </td>
                      <td className="mono-num">{txn.balance_after}</td>
                      <td className="mono-num faint">
                        {new Date(txn.created_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="panel col-12" aria-label={t("wallet.purchase")}>
          <div className="band-title" style={{ padding: "14px 16px 0" }}>
            {t("wallet.purchase")}
          </div>
          <p className="faint" style={{ padding: "0 16px" }}>
            {t("wallet.purchase.body")}
          </p>
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
            <button
              className="btn btn-secondary"
              type="submit"
              disabled={verify.isPending || !receipt.trim()}
            >
              <ShieldCheck size={14} aria-hidden />
              {t("wallet.purchase")}
            </button>
          </form>
        </section>
      </div>
    </div>
  );
}
