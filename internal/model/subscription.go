package model

import (
	"fmt"
	"time"
)

const MonthLayout = "01-2006"

// MonthDate сериализуется/десериализуется как "MM-YYYY".
type MonthDate struct{ time.Time }

func ParseMonthDate(s string) (MonthDate, error) {
	t, err := time.Parse(MonthLayout, s)
	if err != nil {
		return MonthDate{}, fmt.Errorf("дата должна быть в формате MM-YYYY: %w", err)
	}
	return MonthDate{t}, nil
}

func (m MonthDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.Format(MonthLayout) + `"`), nil
}

func (m *MonthDate) UnmarshalJSON(b []byte) error {
	s := string(b)
	if len(s) < 2 {
		return fmt.Errorf("пустая дата")
	}
	t, err := time.Parse(MonthLayout, s[1:len(s)-1])
	if err != nil {
		return fmt.Errorf("дата должна быть в формате MM-YYYY: %w", err)
	}
	m.Time = t
	return nil
}

type Subscription struct {
	ID          int        `db:"id"           json:"id"`
	ServiceName string     `db:"service_name" json:"service_name"`
	Price       int        `db:"price"        json:"price"`
	UserID      string     `db:"user_id"      json:"user_id"`
	StartDate   time.Time  `db:"start_date"   json:"-"`
	EndDate     *time.Time `db:"end_date"     json:"-"`
}

// SubscriptionResponse — то, что отдаём клиенту.
type SubscriptionResponse struct {
	ID          int     `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

func ToResponse(s *Subscription) SubscriptionResponse {
	r := SubscriptionResponse{
		ID:          s.ID,
		ServiceName: s.ServiceName,
		Price:       s.Price,
		UserID:      s.UserID,
		StartDate:   s.StartDate.Format(MonthLayout),
	}
	if s.EndDate != nil {
		v := s.EndDate.Format(MonthLayout)
		r.EndDate = &v
	}
	return r
}

type CreateSubscriptionRequest struct {
	ServiceName string  `json:"service_name"`
	Price       int     `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type UpdateSubscriptionRequest struct {
	ServiceName *string `json:"service_name,omitempty"`
	Price       *int    `json:"price,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

type TotalCostResponse struct {
	Total int `json:"total"`
}
