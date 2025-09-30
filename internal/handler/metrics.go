package handler

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/vvolodya/go-metrics/internal/model"
	"github.com/vvolodya/go-metrics/internal/repository"
)

type Handler struct {
	store repository.MetricsStorage
}

func NewHandler(store repository.MetricsStorage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
		http.Error(w, "Wrong Content-Type Header Value", http.StatusBadRequest)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if parts[1] != "update" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	metricValue := parts[4]
	metricName := parts[3]
	metricType := parts[2]

	if metricName == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch metricType {
	case model.Gauge:
		{
			v, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "Bad value for gauge", http.StatusBadRequest)
				return
			}

			isInf := math.IsInf(v, 0)
			isNan := math.IsNaN(v)
			if isInf || isNan {
				http.Error(w, "Bad value for gauge", http.StatusBadRequest)
				return
			}

			h.store.UpdateGauge(metricName, v)

		}
	case model.Counter:
		{
			v, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "Bad value for counter", http.StatusBadRequest)
				return
			}

			h.store.AddCounter(metricName, v)
		}
	default:
		{
			http.Error(w, "Bad metric type", http.StatusBadRequest)
			return
		}
	}

	newCounterValue, _ := h.store.GetCounter(metricName)
	newGaugeValue, _ := h.store.GetGauge(metricName)
	log.Printf("new counter - %d, new gauge - %f, method - %s, path - %s", newCounterValue, newGaugeValue, r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}
