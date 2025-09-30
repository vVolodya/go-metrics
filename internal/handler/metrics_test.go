package handler

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vvolodya/go-metrics/internal/repository"
)

// small helper to spin handler+store per test
func newTestHandler() (*Handler, repository.MetricsStorage) {
	store := repository.NewMemStorage()
	return NewHandler(store), store
}

func TestUpdateGauge_OK(t *testing.T) {
	h, store := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/temperature/36.6", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want %d, got %d", http.StatusOK, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("content-type: want %q, got %q", "text/plain; charset=utf-8", ct)
	}
	if body := rr.Body.String(); body != "Ok" {
		t.Fatalf("body: want %q, got %q", "Ok", body)
	}

	v, ok := store.GetGauge("temperature")
	if !ok {
		t.Fatalf("GetGauge: want ok==true, got false")
	}
	if math.Abs(v-36.6) > 1e-9 {
		t.Fatalf("GetGauge: want 36.6, got %v", v)
	}
}

func TestUpdateCounter_OK_Sums(t *testing.T) {
	h, store := newTestHandler()

	// +10
	req1 := httptest.NewRequest(http.MethodPost, "/update/counter/requests/10", nil)
	req1.Header.Set("Content-Type", "text/plain")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first status: want %d, got %d", http.StatusOK, rr1.Code)
	}

	// +5
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/requests/5", nil)
	req2.Header.Set("Content-Type", "text/plain")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second status: want %d, got %d", http.StatusOK, rr2.Code)
	}

	v, ok := store.GetCounter("requests")
	if !ok || v != 15 {
		t.Fatalf("GetCounter: want (15, true), got (%d, %v)", v, ok)
	}
}

func TestBadType_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/banana/x/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rr.Code)
	}
}

func TestBadValue_Counter_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/counter/c/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rr.Code)
	}
}

func TestBadValue_Gauge_NaN_Inf_Returns400(t *testing.T) {
	t.Run("NaN", func(t *testing.T) {
		h, _ := newTestHandler()

		req := httptest.NewRequest(http.MethodPost, "/update/gauge/t/NaN", nil)
		req.Header.Set("Content-Type", "text/plain")
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status NaN: want 400, got %d", rr.Code)
		}
	})

	t.Run("Inf", func(t *testing.T) {
		h, _ := newTestHandler()

		req := httptest.NewRequest(http.MethodPost, "/update/gauge/t/Inf", nil)
		req.Header.Set("Content-Type", "text/plain")
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status Inf: want 400, got %d", rr.Code)
		}
	})
}

func TestNoName_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge//36.6", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rr.Code)
	}
}

func TestWrongMethod_Returns405_WithAllowHeader(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/x/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: want 405, got %d", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow header: want %q, got %q", http.MethodPost, allow)
	}
}

func TestWrongPathLen_Returns404(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/x", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rr.Code)
	}
}

func TestContentType_Required_Returns400(t *testing.T) {
	h, _ := newTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/t/1.23", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rr.Code)
	}
}
