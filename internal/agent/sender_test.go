package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/vvolodya/go-metrics/internal/model"
)

type fakeStore2 struct{}

func (fakeStore2) UpdateGauge(name string, v float64)        {}
func (fakeStore2) AddCounter(name string, delta int64) int64 { return 0 }
func (fakeStore2) Snapshot() (map[string]float64, map[string]int64) {
	return map[string]float64{
			"Alloc":  123.45,
			"Custom": 1.23,
		}, map[string]int64{
			"PollCount": 7,
		}
}

func TestHTTPSender_ReportOnce(t *testing.T) {
	var received []string

	// поднимаем фейковый сервер, который собирает вызовы
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = append(received, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	sender := NewHTTPSender(fakeStore2{}, ts.URL)

	err := sender.ReportOnce(context.Background())
	if err != nil {
		t.Fatalf("ReportOnce() returned error: %v", err)
	}

	// Проверяем, что все метрики были отправлены
	wantParts := []string{
		"/update/gauge/Alloc/",
		"/update/gauge/Custom/",
		"/update/counter/PollCount/",
	}

	for _, want := range wantParts {
		found := false
		for _, got := range received {
			if strings.HasPrefix(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected path with prefix %q, but not found in %v", want, received)
		}
	}
}

func TestHTTPSender_ReportOnce_ServerError(t *testing.T) {
	// сервер всегда отвечает 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	sender := NewHTTPSender(fakeStore2{}, ts.URL)

	err := sender.ReportOnce(context.Background())
	if err == nil {
		t.Errorf("expected error on 500 status, got nil")
	}
}

func TestHTTPSender_sendMetric_OK(t *testing.T) {
	var gotMethod, gotContentType, gotPath string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := NewHTTPSender(fakeStore2{}, ts.URL)

	err := s.sendMetric(context.Background(), model.Gauge, "Alloc", "123.45")
	if err != nil {
		t.Fatalf("sendMetric returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method: want POST, got %s", gotMethod)
	}
	if gotContentType != "text/plain" {
		t.Errorf("Content-Type: want %q, got %q", "text/plain", gotContentType)
	}
	wantPath := "/update/gauge/Alloc/123.45"
	if gotPath != wantPath {
		t.Errorf("path: want %q, got %q", wantPath, gotPath)
	}
}

func TestHTTPSender_sendMetric_PathEscaping(t *testing.T) {
	var gotEscapedPath string

	name := "my metric/with space"
	value := "1.23"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEscapedPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := NewHTTPSender(fakeStore2{}, ts.URL)

	if err := s.sendMetric(context.Background(), model.Gauge, name, value); err != nil {
		t.Fatalf("sendMetric returned error: %v", err)
	}

	escaped := url.PathEscape(name)
	want := "/update/gauge/" + escaped + "/" + value
	if gotEscapedPath != want {
		t.Errorf("escaped path: want %q, got %q", want, gotEscapedPath)
	}
}

func TestHTTPSender_sendMetric_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	s := NewHTTPSender(fakeStore2{}, ts.URL)

	err := s.sendMetric(context.Background(), model.Counter, "PollCount", "7")
	if err == nil {
		t.Fatalf("expected error on non-200 status, got nil")
	}
}

func TestHTTPSender_sendMetric_ContextCanceled(t *testing.T) {
	// сервер просто вернёт 200, но мы отменим ctx до запроса
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := NewHTTPSender(fakeStore2{}, ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем до вызова

	err := s.sendMetric(ctx, model.Gauge, "Alloc", "1")
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	// полезно убедиться, что это именно контекстная ошибка
	if !strings.Contains(err.Error(), "context canceled") {
		// оставляем мягкую проверку — реализация может вернуть context.Canceled или обёртку
		t.Logf("returned error: %v", err)
	}
}

func TestHTTPSender_sendMetric_RequestTimeout(t *testing.T) {
	// Смоделируем «медленный» сервер и свой клиент с маленьким таймаутом
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	s := NewHTTPSender(fakeStore2{}, ts.URL)
	// Уменьшим таймаут клиента специально под тест
	s.client.Timeout = 50 * time.Millisecond

	err := s.sendMetric(context.Background(), model.Gauge, "Alloc", "1")
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}
