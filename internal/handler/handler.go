package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"subscriptions/internal/model"
	"subscriptions/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *service.SubscriptionService
	log     *slog.Logger
}

func New(svc *service.SubscriptionService, log *slog.Logger) *Handler {
	return &Handler{service: svc, log: log}
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
	resp, err := h.service.Create(r.Context(), &req)
	if err != nil {
		h.log.Error("create: service", "err", err)
		if errors.Is(err, service.ErrInvalidArgument) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.log.Info("subscription created", "id", resp.ID)
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	sub, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.log.Error("getbyid: service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID := q.Get("user_id")
	limit := 100
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}

	resp, err := h.service.List(r.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("list: service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
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
	if err := h.service.Update(r.Context(), id, &req); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.log.Error("update: service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
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
	if err := h.service.Delete(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.log.Error("delete: service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.log.Info("subscription deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) TotalCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	total, err := h.service.TotalCost(r.Context(), q.Get("user_id"), q.Get("service_name"), q.Get("from"), q.Get("to"))
	if err != nil {
		if errors.Is(err, service.ErrInvalidArgument) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.log.Error("totalcost: service", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
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
