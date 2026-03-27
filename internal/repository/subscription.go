package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"subscriptions/internal/model"

	"github.com/jmoiron/sqlx"
)

type SubscriptionRepo struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

func (r *SubscriptionRepo) Create(ctx context.Context, s *model.Subscription) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.ServiceName, s.Price, s.UserID, s.StartDate, s.EndDate,
	).Scan(&id)
	return id, err
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id int) (*model.Subscription, error) {
	var s model.Subscription
	err := r.db.GetContext(ctx, &s, `SELECT * FROM subscriptions WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepo) List(ctx context.Context, userID string, limit, offset int) ([]model.Subscription, error) {
	var subs []model.Subscription
	query := `SELECT * FROM subscriptions`
	args := []interface{}{}
	if userID != "" {
		query += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	query += ` ORDER BY id LIMIT $2 OFFSET $3`
	args = append(args, limit, offset)
	err := r.db.SelectContext(ctx, &subs, query, args...)
	return subs, err
}

func (r *SubscriptionRepo) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	i := 1
	if req.ServiceName != nil {
		setClauses = append(setClauses, fmt.Sprintf("service_name=$%d", i))
		args = append(args, *req.ServiceName)
		i++
	}
	if req.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price=$%d", i))
		args = append(args, *req.Price)
		i++
	}
	if req.EndDate != nil {
		t, err := time.Parse(model.MonthLayout, *req.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end_date format: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("end_date=$%d", i))
		args = append(args, t)
		i++
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id=$%d", strings.Join(setClauses, ", "), i)
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SubscriptionRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SubscriptionRepo) FindForTotalCost(ctx context.Context, userID, serviceName string, from, to time.Time) ([]model.Subscription, error) {
	query := `SELECT * FROM subscriptions WHERE start_date <= $1 AND (end_date IS NULL OR end_date >= $2)`
	args := []interface{}{to, from}
	if userID != "" {
		query += ` AND user_id=$3`
		args = append(args, userID)
	}
	if serviceName != "" {
		query += ` AND service_name=$4`
		args = append(args, serviceName)
	}
	var subs []model.Subscription
	if err := r.db.SelectContext(ctx, &subs, query, args...); err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *SubscriptionRepo) TotalCost(ctx context.Context, userID, serviceName, from, to string) (int, error) {
	query := `SELECT COALESCE(SUM(price),0) FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	i := 1
	if userID != "" {
		query += fmt.Sprintf(" AND user_id=$%d", i)
		args = append(args, userID)
		i++
	}
	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name=$%d", i)
		args = append(args, serviceName)
		i++
	}
	if from != "" {
		t, err := time.Parse("01-2006", from)
		if err != nil {
			return 0, fmt.Errorf("invalid from format: %w", err)
		}
		query += fmt.Sprintf(" AND start_date>=$%d", i)
		args = append(args, t)
		i++
	}
	if to != "" {
		t, err := time.Parse("01-2006", to)
		if err != nil {
			return 0, fmt.Errorf("invalid to format: %w", err)
		}
		query += fmt.Sprintf(" AND start_date<=$%d", i)
		args = append(args, t)
	}
	var total int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}
