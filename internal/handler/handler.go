package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"subscriptions/internal/model"
	"subscriptions/internal/repository"
)

type Handler struct {
	repo *repository.SubscriptionRepo
	log  *slog.Logger
}

func New(repo *repository.SubscriptionRepo, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/subscriptions", h.Create)
	r.Get("/subscriptions", h.List)
	// /total должен быть зарегистрирован ДО /{id}, иначе chi матчит "total" как id
	r.Get("/subscriptions/total", h.TotalCost)
	r.Get("/subscriptions/{id}", h.GetByID)
	r.Put("/subscriptions/{id}", h.Update)
	r.Delete("/subscriptions/{id}", h.Delete)
	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("create: decode", "err", err)
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.ServiceName == "" || req.Price <= 0 || req.UserID == "" || req.StartDate == "" {
		writeError(w, http.StatusBadRequest, "service_name, price, user_id, start_date are required; price must be > 0")
		return
	}
	startDate, err := time.Parse(model.MonthLayout, req.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "start_date must be MM-YYYY")
		return
	}
	sub := &model.Subscription{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   startDate,
	}
	if req.EndDate != nil {
		t, err := time.Parse(model.MonthLayout, *req.EndDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "end_date must be MM-YYYY")
			return
		}
		sub.EndDate = &t
	}
	id, err := h.repo.Create(r.Context(), sub)
	if err != nil {
		h.log.Error("create: repo", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.log.Info("subscription created", "id", id)
	sub.ID = id
	writeJSON(w, http.StatusCreated, model.ToResponse(sub))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sub, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.log.Error("getbyid: repo", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, model.ToResponse(sub))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	subs, err := h.repo.List(r.Context(), userID)
	if err != nil {
		h.log.Error("list: repo", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	resp := make([]model.SubscriptionResponse, len(subs))
	for i := range subs {
		resp[i] = model.ToResponse(&subs[i])
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req model.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.repo.Update(r.Context(), id, &req); err != nil {
		h.log.Error("update: repo", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.log.Info("subscription updated", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete: repo", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.log.Info("subscription deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) TotalCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	total, err := h.repo.TotalCost(
		r.Context(), // было context.Background() — исправлено
		q.Get("user_id"),
		q.Get("service_name"),
		q.Get("from"),
		q.Get("to"),
	)
	if err != nil {
		h.log.Error("totalcost: repo", "err", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, model.TotalCostResponse{Total: total})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
