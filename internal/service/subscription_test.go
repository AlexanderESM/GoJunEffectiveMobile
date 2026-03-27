package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"subscriptions/internal/model"
)

type fakeRepo struct {
	subs []model.Subscription
}

func (r *fakeRepo) Create(ctx context.Context, s *model.Subscription) (int, error)   { return 0, nil }
func (r *fakeRepo) GetByID(ctx context.Context, id int) (*model.Subscription, error) { return nil, nil }
func (r *fakeRepo) List(ctx context.Context, userID string, limit, offset int) ([]model.Subscription, error) {
	return nil, nil
}
func (r *fakeRepo) Update(ctx context.Context, id int, req *model.UpdateSubscriptionRequest) error {
	return nil
}
func (r *fakeRepo) Delete(ctx context.Context, id int) error { return nil }
func (r *fakeRepo) FindForTotalCost(ctx context.Context, userID, serviceName string, from, to time.Time) ([]model.Subscription, error) {
	result := make([]model.Subscription, 0, len(r.subs))
	for _, s := range r.subs {
		if userID != "" && s.UserID != userID {
			continue
		}
		if serviceName != "" && s.ServiceName != serviceName {
			continue
		}
		// intersection logic must happen in service; this repo returns all candidates
		if s.StartDate.After(to) {
			continue
		}
		if s.EndDate != nil && s.EndDate.Before(from) {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func TestTotalCost(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := &fakeRepo{subs: []model.Subscription{
		{ServiceName: "A", Price: 100, UserID: "u1", StartDate: mustParse("11-2024"), EndDate: ptrTime(mustParse("02-2025"))},
		{ServiceName: "A", Price: 100, UserID: "u1", StartDate: mustParse("03-2025"), EndDate: nil},
		{ServiceName: "A", Price: 100, UserID: "u1", StartDate: mustParse("12-2025"), EndDate: ptrTime(mustParse("01-2026"))},
		{ServiceName: "B", Price: 200, UserID: "u1", StartDate: mustParse("02-2025"), EndDate: ptrTime(mustParse("02-2025"))},
	}}
	service := New(repo, log)
	total, err := service.TotalCost(context.Background(), "u1", "A", "01-2025", "12-2025")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1300 {
		t.Fatalf("expected total %d, got %d", 1300, total)
	}

	// check service name filter
	filtered, err := service.TotalCost(context.Background(), "u1", "B", "01-2025", "12-2025")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if filtered != 200 {
		t.Fatalf("expected total %d, got %d", 200, filtered)
	}

	// invalid args
	_, err = service.TotalCost(context.Background(), "u1", "A", "", "12-2025")
	if err == nil {
		t.Fatalf("expected error for missing from")
	}
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected ErrInvalidArgument, got %v", err)
	}
}

func mustParse(date string) time.Time {
	t, _ := time.Parse(model.MonthLayout, date)
	return t
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
