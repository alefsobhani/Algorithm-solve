package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	etasvc "github.com/example/ridellite/internal/eta/service"
	"github.com/example/ridellite/internal/trip/domain"
)

// HTTP exposes the /v1/eta endpoint.
type HTTP struct {
	svc *etasvc.Service
}

// New creates the handler.
func New(svc *etasvc.Service) *HTTP {
	return &HTTP{svc: svc}
}

// Router builds the chi router.
func (h *HTTP) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/v1/eta", h.estimate)
	return r
}

func (h *HTTP) estimate(w http.ResponseWriter, r *http.Request) {
	pickup := domain.GeoPoint{Lat: parseQueryFloat(r, "pickup_lat"), Lng: parseQueryFloat(r, "pickup_lng")}
	dropoff := domain.GeoPoint{Lat: parseQueryFloat(r, "dropoff_lat"), Lng: parseQueryFloat(r, "dropoff_lng")}
	driverETA, driverID := h.svc.EstimateDriverETA(r.Context(), pickup)
	tripETA := h.svc.EstimateTripETA(r.Context(), pickup, dropoff)

	resp := map[string]any{
		"driver_eta_sec": driverETA.Seconds(),
		"trip_eta_sec":   tripETA.Seconds(),
	}
	if driverID != nil {
		resp["driver_id"] = driverID.String()
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseQueryFloat(r *http.Request, key string) float64 {
	v, _ := strconv.ParseFloat(r.URL.Query().Get(key), 64)
	return v
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
