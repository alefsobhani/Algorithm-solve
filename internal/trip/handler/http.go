package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
	"github.com/example/ridellite/internal/trip/service"
)

// HTTP exposes trip endpoints following the Clean Architecture flow.
type HTTP struct {
	svc *service.Service
}

// NewHTTP constructs a handler.
func NewHTTP(svc *service.Service) *HTTP {
	return &HTTP{svc: svc}
}

// Router builds the chi router with all endpoints and middlewares.
func (h *HTTP) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Post("/v1/trips", h.createTrip)
	r.Get("/v1/trips/{id}", h.getTrip)
	r.Post("/v1/trips/{id}/cancel", h.cancelTrip)
	r.Post("/v1/trips/{id}/start", h.startTrip)
	r.Post("/v1/trips/{id}/complete", h.completeTrip)
	return r
}

type createTripRequest struct {
	RiderID     string          `json:"rider_id"`
	Pickup      domain.GeoPoint `json:"pickup"`
	Dropoff     domain.GeoPoint `json:"dropoff"`
	VehicleType string          `json:"vehicle_type"`
}

func (h *HTTP) createTrip(w http.ResponseWriter, r *http.Request) {
	var payload createTripRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	riderID, err := uuid.Parse(payload.RiderID)
	if err != nil {
		http.Error(w, "invalid rider_id", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.CreateTrip(r.Context(), r.Header.Get("Idempotency-Key"), service.CreateTripRequest{
		RiderID:     riderID,
		Pickup:      payload.Pickup,
		Dropoff:     payload.Dropoff,
		VehicleType: payload.VehicleType,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *HTTP) getTrip(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	trip, err := h.svc.GetTrip(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (h *HTTP) cancelTrip(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	actor := domain.StatusCancelledRider
	if r.URL.Query().Get("actor") == "driver" {
		actor = domain.StatusCancelledDriver
	}
	trip, err := h.svc.CancelTrip(r.Context(), id, actor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (h *HTTP) startTrip(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	trip, err := h.svc.StartTrip(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func (h *HTTP) completeTrip(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var payload struct {
		PriceCents int64 `json:"price_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := h.svc.CompleteTrip(r.Context(), id, payload.PriceCents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
