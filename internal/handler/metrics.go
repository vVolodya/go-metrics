package handler

import (
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vvolodya/go-metrics/internal/model"
	"github.com/vvolodya/go-metrics/internal/repository"
)

type Handler struct {
	store repository.MetricsStorage
}

func NewHandler(store repository.MetricsStorage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) HandlePostMetrics(w http.ResponseWriter, r *http.Request) {
	metricValue := chi.URLParam(r, "value")
	metricName := chi.URLParam(r, "metric")
	metricType := chi.URLParam(r, "type")

	switch metricType {
	case model.Gauge:
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

	case model.Counter:
		v, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Bad value for counter", http.StatusBadRequest)
			return
		}

		h.store.AddCounter(metricName, v)
	default:
		http.Error(w, "Bad metric type", http.StatusBadRequest)
		return
	}

	newCounterValue, _ := h.store.GetCounter(metricName)
	newGaugeValue, _ := h.store.GetGauge(metricName)
	log.Printf("new counter - %d, new gauge - %f, method - %s, path - %s", newCounterValue, newGaugeValue, r.Method, r.URL.Path)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	g, c := h.store.Snapshot()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	io.WriteString(w, "<!doctype html><meta charset='utf-8'><title>Metrics</title><h2>Gauge</h2><ul>")
	gauges := make([]string, 0, len(g))
	for k := range g {
		gauges = append(gauges, k)
	}
	sort.Strings(gauges)
	for _, name := range gauges {
		fmt.Fprintf(w, "<li><code>%s</code>: %v</li>", html.EscapeString(name), g[name])
	}

	io.WriteString(w, "</ul><h2>Counter</h2><ul>")
	counters := make([]string, 0, len(c))
	for k := range c {
		counters = append(counters, k)
	}
	sort.Strings(counters)
	for _, name := range counters {
		fmt.Fprintf(w, "<li><code>%s</code>: %d</li>", html.EscapeString(name), c[name])
	}
	io.WriteString(w, "</ul>")
}

func (h *Handler) HandleMetric(w http.ResponseWriter, r *http.Request) {
	metricName := chi.URLParam(r, "metric")
	metricType := strings.ToLower(chi.URLParam(r, "type"))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	switch metricType {
	case model.Counter:
		if value, ok := h.store.GetCounter(metricName); ok {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, strconv.FormatInt(value, 10))
			return
		}
	case model.Gauge:
		if value, ok := h.store.GetGauge(metricName); ok {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, strconv.FormatFloat(value, 'g', -1, 64))
			return
		}
	default:
		http.Error(w, "Bad metric type", http.StatusNotFound)
		return
	}

	http.NotFound(w, r)
}
