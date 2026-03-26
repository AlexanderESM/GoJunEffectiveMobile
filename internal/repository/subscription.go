package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"subscriptions/internal/model"
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

func (r *SubscriptionRepo) List(ctx context.Context, userID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	query := `SELECT * FROM subscriptions`
	args := []interface{}{}
	if userID != "" {
		query += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	err := r.db.SelectContext(ctx, &subs, query, args...)
	return subs, err
}

func (r *SubscriptionRepo) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) error {
	set := ``
	args := []interface{}{}
	i := 1
	if req.ServiceName != nil {
		set += fmt.Sprintf("service_name=$%d,", i)
		args = append(args, *req.ServiceName)
		i++
	}
	if req.Price != nil {
		set += fmt.Sprintf("price=$%d,", i)
		args = append(args, *req.Price)
		i++
	}
	if req.EndDate != nil {
		t, err := time.Parse("01-2006", *req.EndDate)
		if err != nil {
			return fmt.Errorf("invalid end_date format: %w", err)
		}
		set += fmt.Sprintf("end_date=$%d,", i)
		args = append(args, t)
		i++
	}
	if set == "" {
		return nil
	}
	set = set[:len(set)-1]
	args = append(args, id)
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE subscriptions SET %s WHERE id=$%d`, set, i), args...)
	return err
}

func (r *SubscriptionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=$1`, id)
	return err
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
