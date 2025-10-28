package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vvolodya/go-metrics/internal/repository"
)

func newTestRouter() (http.Handler, *repository.MemStorage) {
	store := repository.NewMemStorage()
	h := NewHandler(store)

	r := chi.NewRouter()
	r.Get("/", h.HandleIndex)
	r.Get("/value/{type}/{metric}", h.HandleMetric)
	r.Post("/update/{type}/{metric}/{value}", h.HandlePostMetrics)

	return r, store
}

func TestHandleIndex_OK(t *testing.T) {
	r, store := newTestRouter()

	store.UpdateGauge("Alloc", 123.45)
	store.AddCounter("Polls", 7)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	s := string(body)

	if !strings.Contains(s, "Alloc") || !strings.Contains(s, "123.45") {
		t.Fatalf("index html does not contain gauge metric: %q", s)
	}
	if !strings.Contains(s, "7<") || !strings.Contains(s, "Polls") {
		t.Fatalf("index html does not contain counter metric: %q", s)
	}
}

func TestHandleMetric_Counter_OK(t *testing.T) {
	r, store := newTestRouter()
	store.AddCounter("RequestsTotal", 10)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/RequestsTotal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if got := string(body); got != "10" {
		t.Fatalf("body = %q, want %q", got, "10")
	}
}

func TestHandleMetric_Gauge_OK(t *testing.T) {
	r, store := newTestRouter()
	store.UpdateGauge("HeapInUse", 42.5)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/HeapInUse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if got := string(body); got != "42.5" {
		t.Fatalf("body = %q, want %q", got, "42.5")
	}
}

func TestHandleMetric_NotFound(t *testing.T) {
	r, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/value/counter/Unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleMetric_BadType(t *testing.T) {
	r, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/value/number/Whatever", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandlePostMetrics_Counter_OK(t *testing.T) {
	r, store := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/counter/Requests/5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, _ := io.ReadAll(resp.Body)
	if got := string(body); got != "Ok" {
		t.Fatalf("body = %q, want %q", got, "Ok")
	}
	if v, ok := store.GetCounter("Requests"); !ok || v != 5 {
		t.Fatalf("counter Requests = %d (ok=%v), want 5,true", v, ok)
	}
}

func TestHandlePostMetrics_Gauge_OK(t *testing.T) {
	r, store := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if v, ok := store.GetGauge("Alloc"); !ok || v != 123.456 {
		t.Fatalf("gauge Alloc = %v (ok=%v), want 123.456,true", v, ok)
	}
}

func TestHandlePostMetrics_BadCounterValue(t *testing.T) {
	r, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/counter/Requests/not-int", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePostMetrics_BadGaugeValue(t *testing.T) {
	r, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-float", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandlePostMetrics_BadType(t *testing.T) {
	r, _ := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/number/Alloc/10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
