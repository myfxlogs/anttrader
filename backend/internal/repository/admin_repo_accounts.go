package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"anttrader/internal/model"
)

type AccountWithUser struct {
	model.MTAccount
	UserEmail    string `json:"user_email" db:"user_email"`
	UserNickname string `json:"user_nickname" db:"user_nickname"`
}

func (r *AdminRepository) ListAccounts(ctx context.Context, params *model.AccountListParams) ([]*AccountWithUser, int64, error) {
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

	countQuery := `
		SELECT COUNT(*) FROM mt_accounts ma
		JOIN users u ON ma.user_id = u.id
		WHERE 1=1
	`
	query := `
		SELECT ma.*, u.email as user_email, u.nickname as user_nickname
		FROM mt_accounts ma
		JOIN users u ON ma.user_id = u.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if params.Search != "" {
		countQuery += fmt.Sprintf(" AND (ma.login ILIKE $%d OR u.email ILIKE $%d OR u.nickname ILIKE $%d)", argIndex, argIndex, argIndex)
		query += fmt.Sprintf(" AND (ma.login ILIKE $%d OR u.email ILIKE $%d OR u.nickname ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.Status != "" {
		countQuery += fmt.Sprintf(" AND ma.account_status = $%d", argIndex)
		query += fmt.Sprintf(" AND ma.account_status = $%d", argIndex)
		args = append(args, params.Status)
		argIndex++
	}

	if params.MTType != "" {
		countQuery += fmt.Sprintf(" AND ma.mt_type = $%d", argIndex)
		query += fmt.Sprintf(" AND ma.mt_type = $%d", argIndex)
		args = append(args, params.MTType)
		argIndex++
	}

	if params.UserID != "" {
		countQuery += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		query += fmt.Sprintf(" AND ma.user_id = $%d", argIndex)
		args = append(args, params.UserID)
		argIndex++
	}

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY ma.created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []*AccountWithUser
	for rows.Next() {
		var a AccountWithUser
		err := rows.Scan(
			&a.ID, &a.UserID, &a.MTType, &a.BrokerCompany, &a.BrokerServer,
			&a.BrokerHost, &a.Login, &a.Password, &a.Alias, &a.IsDisabled,
			&a.Balance, &a.Credit, &a.Equity, &a.Margin, &a.FreeMargin,
			&a.MarginLevel, &a.Leverage, &a.Currency, &a.AccountMethod,
			&a.IsInvestor, &a.AccountStatus, &a.StreamStatus, &a.MTToken,
			&a.LastError, &a.LastConnectedAt, &a.LastCheckedAt,
			&a.CreatedAt, &a.UpdatedAt, &a.AccountType,
			&a.UserEmail, &a.UserNickname,
		)
		if err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, &a)
	}

	return accounts, total, nil
}

func (r *AdminRepository) SetAccountStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE mt_accounts SET account_status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	return nil
}

func (r *AdminRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountWithUser, error) {
	query := `
		SELECT ma.*, u.email as user_email, u.nickname as user_nickname
		FROM mt_accounts ma
		JOIN users u ON ma.user_id = u.id
		WHERE ma.id = $1
	`
	var a AccountWithUser
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.UserID, &a.MTType, &a.BrokerCompany, &a.BrokerServer,
		&a.BrokerHost, &a.Login, &a.Password, &a.Alias, &a.IsDisabled,
		&a.Balance, &a.Credit, &a.Equity, &a.Margin, &a.FreeMargin,
		&a.MarginLevel, &a.Leverage, &a.Currency, &a.AccountMethod,
		&a.IsInvestor, &a.AccountStatus, &a.StreamStatus, &a.MTToken,
		&a.LastError, &a.LastConnectedAt, &a.LastCheckedAt,
		&a.CreatedAt, &a.UpdatedAt,
		&a.UserEmail, &a.UserNickname,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &a, nil
}
