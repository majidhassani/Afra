package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "casemind/pkg/errors"
)

type Repository interface {
	GetOrCreate(ctx context.Context, userID uuid.UUID) (*Wallet, error)
	// Reserve atomically moves amount from balance to reserved_balance and
	// records an active reservation. Fails with conflict on insufficient funds.
	Reserve(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, action string, amount int) (*Reservation, error)
	// Settle consumes an active reservation: reserved coins are spent and a
	// transaction is recorded.
	Settle(ctx context.Context, res *Reservation, metadata map[string]any) (*Transaction, error)
	// Release returns an active reservation's coins to the balance.
	Release(ctx context.Context, res *Reservation) error
	// Credit adds coins and records a transaction.
	Credit(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, txType string, amount int, metadata map[string]any) (*Transaction, error)
	Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]Transaction, error)
	Pricing(ctx context.Context) (map[string]int, error)
	InsertUsageLog(ctx context.Context, log *UsageLog) error
	CountAdClaimsSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)
	InsertAdClaim(ctx context.Context, userID uuid.UUID, coins int) error
	InsertReceipt(ctx context.Context, userID uuid.UUID, platform, productID, receiptHash string, coins int, status string) error
	// SpentAndEarned sums negative and positive transaction amounts.
	SpentAndEarned(ctx context.Context, userID uuid.UUID) (spent int, earned int, err error)
}

type PGRepository struct{ pool *pgxpool.Pool }

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) GetOrCreate(ctx context.Context, userID uuid.UUID) (*Wallet, error) {
	w := &Wallet{UserID: userID}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, balance, reserved_balance, created_at, updated_at
		 FROM wallets WHERE user_id = $1`, userID,
	).Scan(&w.ID, &w.UserID, &w.Balance, &w.ReservedBalance, &w.CreatedAt, &w.UpdatedAt)
	if err == nil {
		return w, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.Internal(err, "get wallet")
	}
	// First access: create with the starting balance and record the grant.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.Internal(err, "begin wallet create")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx,
		`INSERT INTO wallets (user_id, balance) VALUES ($1, $2)
		 ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
		 RETURNING id, user_id, balance, reserved_balance, created_at, updated_at`,
		userID, StartingBalance,
	).Scan(&w.ID, &w.UserID, &w.Balance, &w.ReservedBalance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "create wallet")
	}
	if w.CreatedAt.Equal(w.UpdatedAt) { // freshly inserted
		if _, err := tx.Exec(ctx,
			`INSERT INTO wallet_transactions (wallet_id, user_id, type, amount, balance_after, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			w.ID, userID, TxAdminAdjustment, StartingBalance, w.Balance,
			[]byte(`{"reason":"starting_balance"}`)); err != nil {
			return nil, apperrors.Internal(err, "record starting balance")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.Internal(err, "commit wallet create")
	}
	return w, nil
}

func (r *PGRepository) Reserve(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, action string, amount int) (*Reservation, error) {
	if _, err := r.GetOrCreate(ctx, userID); err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.Internal(err, "begin reserve")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var walletID uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE wallets
		 SET balance = balance - $2, reserved_balance = reserved_balance + $2, updated_at = now()
		 WHERE user_id = $1 AND balance >= $2
		 RETURNING id`, userID, amount,
	).Scan(&walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.Conflict("insufficient_balance", "not enough coins for this action")
	}
	if err != nil {
		return nil, apperrors.Internal(err, "reserve coins")
	}

	res := &Reservation{WalletID: walletID, UserID: userID, MissionID: missionID, ActionType: action, Amount: amount, Status: ReservationActive}
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_reservations (wallet_id, user_id, mission_id, action_type, amount)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		walletID, userID, missionID, action, amount,
	).Scan(&res.ID, &res.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "insert reservation")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.Internal(err, "commit reserve")
	}
	return res, nil
}

func (r *PGRepository) Settle(ctx context.Context, res *Reservation, metadata map[string]any) (*Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.Internal(err, "begin settle")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`UPDATE wallet_reservations SET status = $2, updated_at = now()
		 WHERE id = $1 AND status = $3`, res.ID, ReservationSettled, ReservationActive)
	if err != nil {
		return nil, apperrors.Internal(err, "settle reservation")
	}
	if tag.RowsAffected() == 0 {
		return nil, apperrors.Conflict("reservation_not_active", "reservation already settled or released")
	}

	var balance int
	err = tx.QueryRow(ctx,
		`UPDATE wallets SET reserved_balance = reserved_balance - $2, updated_at = now()
		 WHERE id = $1 AND reserved_balance >= $2
		 RETURNING balance`, res.WalletID, res.Amount,
	).Scan(&balance)
	if err != nil {
		return nil, apperrors.Internal(err, "consume reserved coins")
	}

	txn := &Transaction{WalletID: res.WalletID, UserID: res.UserID, MissionID: res.MissionID,
		Type: res.ActionType, Amount: -res.Amount, BalanceAfter: balance, Metadata: mustJSON(metadata)}
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (wallet_id, user_id, mission_id, type, amount, balance_after, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		txn.WalletID, txn.UserID, txn.MissionID, txn.Type, txn.Amount, txn.BalanceAfter, txn.Metadata,
	).Scan(&txn.ID, &txn.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "insert settle transaction")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.Internal(err, "commit settle")
	}
	res.Status = ReservationSettled
	return txn, nil
}

func (r *PGRepository) Release(ctx context.Context, res *Reservation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperrors.Internal(err, "begin release")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`UPDATE wallet_reservations SET status = $2, updated_at = now()
		 WHERE id = $1 AND status = $3`, res.ID, ReservationReleased, ReservationActive)
	if err != nil {
		return apperrors.Internal(err, "release reservation")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.Conflict("reservation_not_active", "reservation already settled or released")
	}
	if _, err := tx.Exec(ctx,
		`UPDATE wallets SET balance = balance + $2, reserved_balance = reserved_balance - $2, updated_at = now()
		 WHERE id = $1 AND reserved_balance >= $2`, res.WalletID, res.Amount); err != nil {
		return apperrors.Internal(err, "return reserved coins")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.Internal(err, "commit release")
	}
	res.Status = ReservationReleased
	return nil
}

func (r *PGRepository) Credit(ctx context.Context, userID uuid.UUID, missionID *uuid.UUID, txType string, amount int, metadata map[string]any) (*Transaction, error) {
	w, err := r.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.Internal(err, "begin credit")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var balance int
	err = tx.QueryRow(ctx,
		`UPDATE wallets SET balance = balance + $2, updated_at = now() WHERE id = $1
		 RETURNING balance`, w.ID, amount,
	).Scan(&balance)
	if err != nil {
		return nil, apperrors.Internal(err, "credit coins")
	}
	txn := &Transaction{WalletID: w.ID, UserID: userID, MissionID: missionID,
		Type: txType, Amount: amount, BalanceAfter: balance, Metadata: mustJSON(metadata)}
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (wallet_id, user_id, mission_id, type, amount, balance_after, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		txn.WalletID, txn.UserID, txn.MissionID, txn.Type, txn.Amount, txn.BalanceAfter, txn.Metadata,
	).Scan(&txn.ID, &txn.CreatedAt)
	if err != nil {
		return nil, apperrors.Internal(err, "insert credit transaction")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.Internal(err, "commit credit")
	}
	return txn, nil
}

func (r *PGRepository) Transactions(ctx context.Context, userID uuid.UUID, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, wallet_id, user_id, mission_id, type, amount, balance_after, metadata, created_at
		 FROM wallet_transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit)
	if err != nil {
		return nil, apperrors.Internal(err, "list transactions")
	}
	defer rows.Close()
	items := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.WalletID, &t.UserID, &t.MissionID, &t.Type, &t.Amount,
			&t.BalanceAfter, &t.Metadata, &t.CreatedAt); err != nil {
			return nil, apperrors.Internal(err, "scan transaction")
		}
		items = append(items, t)
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate transactions")
	}
	return items, nil
}

func (r *PGRepository) Pricing(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT action_type, coins FROM pricing_rules WHERE active`)
	if err != nil {
		return nil, apperrors.Internal(err, "load pricing")
	}
	defer rows.Close()
	pricing := map[string]int{}
	for rows.Next() {
		var action string
		var coins int
		if err := rows.Scan(&action, &coins); err != nil {
			return nil, apperrors.Internal(err, "scan pricing rule")
		}
		pricing[action] = coins
	}
	if rows.Err() != nil {
		return nil, apperrors.Internal(rows.Err(), "iterate pricing rules")
	}
	return pricing, nil
}

func (r *PGRepository) InsertUsageLog(ctx context.Context, log *UsageLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO llm_usage_logs
		 (user_id, mission_id, agent_name, action_type, model, input_tokens, output_tokens, estimated_cost, coins_charged, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		log.UserID, log.MissionID, log.AgentName, log.ActionType, log.Model,
		log.InputTokens, log.OutputTokens, log.EstimatedCost, log.CoinsCharged, log.Status)
	if err != nil {
		return apperrors.Internal(err, "insert llm usage log")
	}
	return nil
}

func (r *PGRepository) CountAdClaimsSince(ctx context.Context, userID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM rewarded_ads WHERE user_id = $1 AND claimed_at >= $2`,
		userID, since).Scan(&n)
	if err != nil {
		return 0, apperrors.Internal(err, "count ad claims")
	}
	return n, nil
}

func (r *PGRepository) InsertAdClaim(ctx context.Context, userID uuid.UUID, coins int) error {
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO rewarded_ads (user_id, coins) VALUES ($1, $2)`, userID, coins); err != nil {
		return apperrors.Internal(err, "insert ad claim")
	}
	return nil
}

func (r *PGRepository) InsertReceipt(ctx context.Context, userID uuid.UUID, platform, productID, receiptHash string, coins int, status string) error {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO purchase_receipts (user_id, platform, product_id, receipt_hash, coins, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (receipt_hash) DO NOTHING`,
		userID, platform, productID, receiptHash, coins, status)
	if err != nil {
		return apperrors.Internal(err, "insert purchase receipt")
	}
	if tag.RowsAffected() == 0 {
		return apperrors.Conflict("receipt_already_used", "this receipt has already been redeemed")
	}
	return nil
}

func (r *PGRepository) SpentAndEarned(ctx context.Context, userID uuid.UUID) (int, int, error) {
	var spent, earned int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0),
		        COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0)
		 FROM wallet_transactions WHERE user_id = $1`, userID).Scan(&spent, &earned)
	if err != nil {
		return 0, 0, apperrors.Internal(err, "sum transactions")
	}
	return spent, earned, nil
}

func mustJSON(m map[string]any) json.RawMessage {
	if m == nil {
		return []byte(`{}`)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}
