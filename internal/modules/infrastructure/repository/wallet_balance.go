package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
)

type WalletBalanceRepository struct {
	db *gorm.DB
}

func NewWalletBalanceRepository(db *gorm.DB) *WalletBalanceRepository {
	return &WalletBalanceRepository{
		db: db,
	}
}

func (r *WalletBalanceRepository) Create(ctx context.Context, snapshot *model.WalletBalanceSnapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}

	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now().UTC()
	}

	if snapshot.SnapshotAt.IsZero() {
		snapshot.SnapshotAt = time.Now().UTC()
	}

	if err := r.db.WithContext(ctx).Create(snapshot).Error; err != nil {
		return err
	}

	return nil
}

func (r *WalletBalanceRepository) GetLatest(ctx context.Context, chain model.Chain, walletAddress string) (*model.WalletBalanceSnapshot, error) {
	snapshot := &model.WalletBalanceSnapshot{}
	if err := r.db.WithContext(ctx).Where("chain = ? AND wallet_address = ?", chain, walletAddress).Order("snapshot_at DESC").First(snapshot).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return snapshot, nil
}

func (r *WalletBalanceRepository) GetHistory(ctx context.Context, chain model.Chain, walletAddress string, startTime, endTime time.Time, limit int) ([]*model.WalletBalanceSnapshot, error) {
	if limit <= 0 {
		limit = 100
	}

	var snapshots []*model.WalletBalanceSnapshot
	if err := r.db.WithContext(ctx).
		Where("chain = ? AND wallet_address = ? AND snapshot_at >= ? AND snapshot_at <= ?", chain, walletAddress, startTime, endTime).
		Order("snapshot_at DESC").
		Limit(limit).
		Find(&snapshots).Error; err != nil {
		return nil, err
	}

	return snapshots, nil
}

func (r *WalletBalanceRepository) MarkAlertSent(ctx context.Context, snapshotID string) error {
	if err := r.db.WithContext(ctx).Model(&model.WalletBalanceSnapshot{}).Where("id = ?", snapshotID).Updates(map[string]interface{}{
		"alert_sent":    true,
		"alert_sent_at": time.Now().UTC(),
	}).Error; err != nil {
		return err
	}

	return nil
}

func (r *WalletBalanceRepository) GetPendingAlerts(ctx context.Context) ([]*model.WalletBalanceSnapshot, error) {
	var snapshots []*model.WalletBalanceSnapshot
	if err := r.db.WithContext(ctx).
		Where("(is_below_min_threshold = ? OR is_above_max_threshold = ?) AND alert_sent = ?", true, true, false).
		Order("snapshot_at DESC").
		Find(&snapshots).Error; err != nil {
		return nil, err
	}

	return snapshots, nil
}
