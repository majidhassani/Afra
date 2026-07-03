// Package wallet owns coins: balances, reservations, transactions, pricing,
// LLM usage logs, rewarded ads, and purchase receipts. Every paid AI action
// in the backend must pass through the WalletGuard (estimate -> reserve ->
// run -> settle/refund). Prices are always resolved server-side.
package wallet

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Transaction types.
const (
	TxPurchase        = "purchase"
	TxRewardedAd      = "rewarded_ad"
	TxDailyBonus      = "daily_bonus"
	TxMissionReward   = "mission_reward"
	TxRefund          = "refund"
	TxAdminAdjustment = "admin_adjustment"
)

// Paid action types (must match pricing_rules seeds).
const (
	ActionMissionStart   = "mission_start"
	ActionCharacterChat  = "character_chat"
	ActionAIGuidance     = "ai_guidance"
	ActionClueExplain    = "clue_explain"
	ActionClueInspect    = "clue_inspect"
	ActionLocationSearch = "location_search"
	ActionLocationAsk    = "location_ask"
	ActionAdvanceTime    = "advance_time"
	ActionFinalJudgment  = "final_judgment"
)

// Reservation statuses.
const (
	ReservationActive   = "active"
	ReservationSettled  = "settled"
	ReservationReleased = "released"
)

// StartingBalance is granted when a wallet is first created.
const StartingBalance = 500

// RewardedAdCoins / RewardedAdDailyLimit control the ad reward faucet.
const (
	RewardedAdCoins      = 25
	RewardedAdDailyLimit = 5
)

type Wallet struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Balance         int       `json:"balance"`
	ReservedBalance int       `json:"reserved_balance"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Transaction struct {
	ID           uuid.UUID       `json:"id"`
	WalletID     uuid.UUID       `json:"wallet_id"`
	UserID       uuid.UUID       `json:"user_id"`
	MissionID    *uuid.UUID      `json:"mission_id,omitempty"`
	Type         string          `json:"type"`
	Amount       int             `json:"amount"`
	BalanceAfter int             `json:"balance_after"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Reservation struct {
	ID         uuid.UUID
	WalletID   uuid.UUID
	UserID     uuid.UUID
	MissionID  *uuid.UUID
	ActionType string
	Amount     int
	Status     string
	CreatedAt  time.Time
}

type UsageLog struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	MissionID     *uuid.UUID
	AgentName     string
	ActionType    string
	Model         string
	InputTokens   int
	OutputTokens  int
	EstimatedCost float64
	CoinsCharged  int
	Status        string
	CreatedAt     time.Time
}

// Cost is embedded in AI action responses so the client can display charges.
type Cost struct {
	CoinsCharged int `json:"coins_charged"`
}
