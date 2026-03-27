package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"subscriptions/internal/model"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
)

type subscriptionRepo interface {
	Create(ctx context.Context, s *model.Subscription) (int, error)
	GetByID(ctx context.Context, id int) (*model.Subscription, error)
	List(ctx context.Context, userID string, limit, offset int) ([]model.Subscription, error)
	Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) error
	Delete(ctx context.Context, id int) error
	FindForTotalCost(ctx context.Context, userID, serviceName string, from, to time.Time) ([]model.Subscription, error)
}

type SubscriptionService struct {
	repo subscriptionRepo
	log  *slog.Logger
}

func New(repo subscriptionRepo, log *slog.Logger) *SubscriptionService {
	return &SubscriptionService{repo: repo, log: log}
}

func (s *SubscriptionService) Create(ctx context.Context, req *model.CreateSubscriptionRequest) (*model.SubscriptionResponse, error) {
	if req.ServiceName == "" || req.Price <= 0 || req.UserID == "" || req.StartDate == "" {
		return nil, fmt.Errorf("required fields: %w", ErrInvalidArgument)
	}
	start, err := time.Parse(model.MonthLayout, req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("start_date must be MM-YYYY: %w", ErrInvalidArgument)
	}
	sub := &model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   start,
	}
	if req.EndDate != nil {
		end, err := time.Parse(model.MonthLayout, *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("end_date must be MM-YYYY: %w", ErrInvalidArgument)
		}
		sub.EndDate = &end
	}
	id, err := s.repo.Create(ctx, sub)
	if err != nil {
		return nil, err
	}
	sub.ID = id
	resp := model.ToResponse(sub)
	return &resp, nil
}

func (s *SubscriptionService) GetByID(ctx context.Context, id int) (*model.SubscriptionResponse, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	resp := model.ToResponse(sub)
	return &resp, nil
}

func (s *SubscriptionService) List(ctx context.Context, userID string, limit, offset int) ([]model.SubscriptionResponse, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	subs, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	resp := make([]model.SubscriptionResponse, len(subs))
	for i := range subs {
		resp[i] = model.ToResponse(&subs[i])
	}
	return resp, nil
}

func (s *SubscriptionService) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) error {
	if req.ServiceName == nil && req.Price == nil && req.EndDate == nil {
		return nil
	}
	err := s.repo.Update(ctx, id, req)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *SubscriptionService) Delete(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *SubscriptionService) TotalCost(ctx context.Context, userID, serviceName, from, to string) (int, error) {
	if from == "" || to == "" {
		return 0, fmt.Errorf("from/to are required: %w", ErrInvalidArgument)
	}
	fromDate, err := time.Parse(model.MonthLayout, from)
	if err != nil {
		return 0, fmt.Errorf("from invalid: %w", ErrInvalidArgument)
	}
	toDate, err := time.Parse(model.MonthLayout, to)
	if err != nil {
		return 0, fmt.Errorf("to invalid: %w", ErrInvalidArgument)
	}
	if fromDate.After(toDate) {
		return 0, fmt.Errorf("from must be <= to: %w", ErrInvalidArgument)
	}
	subs, err := s.repo.FindForTotalCost(ctx, userID, serviceName, fromDate, toDate)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, sub := range subs {
		start := maxMonth(sub.StartDate, fromDate)
		end := toDate
		if sub.EndDate != nil && sub.EndDate.Before(end) {
			end = *sub.EndDate
		}
		if end.Before(start) {
			continue
		}
		months := monthsInclusive(start, end)
		total += sub.Price * months
	}
	return total, nil
}

func maxMonth(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func monthsInclusive(start, end time.Time) int {
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months + 1
}
