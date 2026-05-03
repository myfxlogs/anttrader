package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

type UserWithAccounts struct {
	model.User
	MTAccountCount int `json:"mt_account_count" db:"mt_account_count"`
}

func (r *AdminRepository) ListUsers(ctx context.Context, params *model.UserListParams) ([]*UserWithAccounts, int64, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	offset := (params.Page - 1) * params.PageSize

	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	query := `
		SELECT u.*, COUNT(ma.id) as mt_account_count
		FROM users u
		LEFT JOIN mt_accounts ma ON u.id = ma.user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if params.Search != "" {
		countQuery += fmt.Sprintf(" AND (email ILIKE $%d OR nickname ILIKE $%d)", argIndex, argIndex)
		query += fmt.Sprintf(" AND (u.email ILIKE $%d OR u.nickname ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.Status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		query += fmt.Sprintf(" AND u.status = $%d", argIndex)
		args = append(args, params.Status)
		argIndex++
	}

	if params.Role != "" {
		countQuery += fmt.Sprintf(" AND role = $%d", argIndex)
		query += fmt.Sprintf(" AND u.role = $%d", argIndex)
		args = append(args, params.Role)
		argIndex++
	}

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" GROUP BY u.id ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*UserWithAccounts
	for rows.Next() {
		var u UserWithAccounts
		err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Nickname, &u.Avatar,
			&u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
			&u.MTAccountCount,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}

	return users, total, nil
}

func (r *AdminRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, avatar, role, status,
			   last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	user := &model.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Nickname, &user.Avatar,
		&user.Role, &user.Status, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *AdminRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (email, password_hash, nickname, avatar, role, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.Nickname, user.Avatar, user.Role, user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *AdminRepository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET nickname = $2, avatar = $3, role = $4, status = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query,
		user.ID, user.Nickname, user.Avatar, user.Role, user.Status,
	).Scan(&user.UpdatedAt)
}

func (r *AdminRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AdminRepository) SetUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE users SET status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AdminRepository) ResetUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *AdminRepository) HasPermission(ctx context.Context, role, permissionCode string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM role_permissions rp
			JOIN permissions p ON rp.permission_id = p.id
			WHERE rp.role = $1 AND p.code = $2
		)
	`
	var hasPermission bool
	err := r.db.QueryRow(ctx, query, role, permissionCode).Scan(&hasPermission)
	return hasPermission, err
}

func (r *AdminRepository) GetTradingSummary(ctx context.Context, startDate, endDate string) (*model.TradingSummary, error) {
	summary := &model.TradingSummary{}
	summary.Period.StartDate = startDate
	summary.Period.EndDate = endDate
	summary.ByPlatform = make(map[string]struct {
		Accounts int64   `json:"accounts"`
		Orders   int64   `json:"orders"`
		Volume   float64 `json:"volume"`
	})

	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&summary.Overview.TotalUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active'`).Scan(&summary.Overview.ActiveUsers)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts`).Scan(&summary.Overview.TotalAccounts)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM mt_accounts WHERE account_status = 'connected'`).Scan(&summary.Overview.ConnectedAccounts)
	if err != nil {
		return nil, err
	}

	timeCondition := ""
	args := []interface{}{}
	if startDate != "" && endDate != "" {
		timeCondition = "WHERE close_time >= $1 AND close_time <= $2"
		args = append(args, startDate+" 00:00:00", endDate+" 23:59:59")
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_orders,
			COUNT(CASE WHEN close_time IS NOT NULL THEN 1 END) as closed_orders,
			COUNT(CASE WHEN close_time IS NULL THEN 1 END) as pending_orders,
			COALESCE(SUM(volume), 0) as total_volume,
			COALESCE(SUM(CASE WHEN profit > 0 THEN profit ELSE 0 END), 0) as total_profit,
			COALESCE(SUM(CASE WHEN profit < 0 THEN profit ELSE 0 END), 0) as total_loss,
			COALESCE(SUM(profit), 0) as net_profit
		FROM trade_records %s
	`, timeCondition)

	err = r.db.QueryRow(ctx, query, args...).Scan(
		&summary.Trading.TotalOrders, &summary.Trading.ClosedOrders, &summary.Trading.PendingOrders,
		&summary.Trading.TotalVolume, &summary.Trading.TotalProfit, &summary.Trading.TotalLoss, &summary.Trading.NetProfit,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	platformQuery := `
		SELECT
			CASE WHEN ma.mt_type = 'MT4' THEN 'MT4' ELSE 'MT5' END as platform,
			COUNT(DISTINCT tr.account_id) as accounts,
			COUNT(*) as orders,
			COALESCE(SUM(tr.volume), 0) as volume
		FROM trade_records tr
		JOIN mt_accounts ma ON tr.account_id = ma.id
		GROUP BY ma.mt_type
	`
	rows, err := r.db.Query(ctx, platformQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var platform string
		var data struct {
			Accounts int64   `json:"accounts"`
			Orders   int64   `json:"orders"`
			Volume   float64 `json:"volume"`
		}
		err := rows.Scan(&platform, &data.Accounts, &data.Orders, &data.Volume)
		if err != nil {
			return nil, err
		}
		summary.ByPlatform[platform] = data
	}

	return summary, nil
}
