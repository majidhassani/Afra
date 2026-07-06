package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	apperrors "casemind/pkg/errors"
	"casemind/pkg/validator"
)

// Products purchasable with real money. Receipt validation against the store
// is a placeholder — receipts are recorded and deduplicated server-side.
var products = map[string]int{
	"coins_small":  200,
	"coins_medium": 600,
	"coins_large":  1500,
}

type Service struct {
	repo Repository
	// demoPurchases enables the mock purchase-verification and rewarded-ad
	// endpoints. When false (production without an explicit opt-in) these
	// endpoints are refused. Real balance/pricing/AI charging is unaffected.
	demoPurchases bool
}

func NewService(repo Repository, demoPurchases bool) *Service {
	return &Service{repo: repo, demoPurchases: demoPurchases}
}

// DemoPurchasesEnabled reports whether the mock purchase/ad paths are active.
func (s *Service) DemoPurchasesEnabled() bool { return s.demoPurchases }

// CoinPack is one purchasable coin bundle in the store catalog.
type CoinPack struct {
	ProductID string `json:"product_id"`
	Coins     int    `json:"coins"`
}

// Catalog returns the purchasable coin packs, cheapest first.
func (s *Service) Catalog() []CoinPack {
	packs := []CoinPack{
		{ProductID: "coins_small", Coins: products["coins_small"]},
		{ProductID: "coins_medium", Coins: products["coins_medium"]},
		{ProductID: "coins_large", Coins: products["coins_large"]},
	}
	return packs
}

// RewardedAdCoinValue exposes the fixed rewarded-ad payout for the UI.
func (s *Service) RewardedAdCoinValue() int { return RewardedAdCoins }

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*Wallet, error) {
	return s.repo.GetOrCreate(ctx, userID)
}

func (s *Service) Balance(ctx context.Context, userID uuid.UUID) (int, error) {
	w, err := s.repo.GetOrCreate(ctx, userID)
	if err != nil {
		return 0, err
	}
	return w.Balance, nil
}

func (s *Service) Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]Transaction, error) {
	return s.repo.Transactions(ctx, userID, limit)
}

func (s *Service) Pricing(ctx context.Context) (map[string]int, error) {
	return s.repo.Pricing(ctx)
}

// PriceOf resolves the server-side price for a paid action.
func (s *Service) PriceOf(ctx context.Context, action string) (int, error) {
	pricing, err := s.repo.Pricing(ctx)
	if err != nil {
		return 0, err
	}
	price, ok := pricing[action]
	if !ok {
		return 0, apperrors.Internal(nil, "no pricing rule for action: "+action)
	}
	return price, nil
}

func (s *Service) Reserve(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, action string) (*Reservation, error) {
	price, err := s.PriceOf(ctx, action)
	if err != nil {
		return nil, err
	}
	return s.repo.Reserve(ctx, userID, missionID, action, price)
}

func (s *Service) Settle(ctx context.Context, res *Reservation, metadata map[string]any) (*Transaction, error) {
	return s.repo.Settle(ctx, res, metadata)
}

func (s *Service) Release(ctx context.Context, res *Reservation) error {
	return s.repo.Release(ctx, res)
}

func (s *Service) Credit(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, txType string, amount int, metadata map[string]any) (*Transaction, error) {
	if amount <= 0 {
		return nil, apperrors.Invalid("invalid_amount", "credit amount must be positive")
	}
	return s.repo.Credit(ctx, userID, missionID, txType, amount, metadata)
}

func (s *Service) SpentAndEarned(ctx context.Context, userID uuid.UUID) (int, int, error) {
	return s.repo.SpentAndEarned(ctx, userID)
}

func (s *Service) InsertUsageLog(ctx context.Context, log *UsageLog) error {
	return s.repo.InsertUsageLog(ctx, log)
}

// ClaimRewardedAd credits the ad reward, enforcing the server-side daily cap.
// Real ad-network callback validation is a placeholder; double-claim
// protection is enforced here.
func (s *Service) ClaimRewardedAd(ctx context.Context, userID uuid.UUID) (*Transaction, error) {
	if !s.demoPurchases {
		return nil, apperrors.Conflict("demo_disabled", "rewarded ads are not available in this environment")
	}
	since := time.Now().UTC().Truncate(24 * time.Hour)
	claims, err := s.repo.CountAdClaimsSince(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	if claims >= RewardedAdDailyLimit {
		return nil, apperrors.Conflict("ad_limit_reached", "daily rewarded ad limit reached")
	}
	if err := s.repo.InsertAdClaim(ctx, userID, RewardedAdCoins); err != nil {
		return nil, err
	}
	return s.repo.Credit(ctx, userID, nil, TxRewardedAd, RewardedAdCoins, map[string]any{"source": "rewarded_ad"})
}

// VerifyPurchase records the receipt (deduplicated by hash) and credits the
// product's coins. Store-side receipt verification is a placeholder hook.
func (s *Service) VerifyPurchase(ctx context.Context, userID uuid.UUID, platform, productID, receipt string) (*Transaction, error) {
	if !s.demoPurchases {
		return nil, apperrors.Conflict("demo_disabled", "demo purchases are disabled in this environment")
	}
	if err := validator.New().
		Required("platform", platform).OneOf("platform", platform, "ios", "android").
		Required("product_id", productID).
		Required("receipt", receipt).
		Err(); err != nil {
		return nil, err
	}
	coins, ok := products[productID]
	if !ok {
		return nil, apperrors.Invalid("unknown_product", "unknown product id")
	}
	sum := sha256.Sum256([]byte(receipt))
	hash := hex.EncodeToString(sum[:])
	if err := s.repo.InsertReceipt(ctx, userID, platform, productID, hash, coins, "verified"); err != nil {
		return nil, err
	}
	return s.repo.Credit(ctx, userID, nil, TxPurchase, coins, map[string]any{
		"platform": platform, "product_id": productID,
	})
}
