package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrConfigNotFound   = errors.New("config not found")
	ErrLogNotFound      = errors.New("log not found")
	ErrPermissionDenied = errors.New("permission denied")
)

type AdminRepository struct {
	db     *pgxpool.Pool
	sqlxDB *sqlx.DB
}

func NewAdminRepository(db *pgxpool.Pool, sqlxDB *sqlx.DB) *AdminRepository {
	return &AdminRepository{db: db, sqlxDB: sqlxDB}
}

func (r *AdminRepository) GetDashboardStats(ctx context.Context) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}

	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&stats.TotalAccounts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status = 'connected'`).Scan(&stats.OnlineAccounts)
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayTrades)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(volume), 0) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayVolume)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(profit), 0) FROM trade_records 
		WHERE DATE(close_time) = $1
	`, today).Scan(&stats.TodayProfit)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	stats.SystemLoad = 0.0

	return stats, nil
}
