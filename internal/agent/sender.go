package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vvolodya/go-metrics/internal/model"
)

type Sender interface {
	ReportOnce(ctx context.Context) error
}

type HTTPSender struct {
	store   LocalStorage
	baseURL string
	Client  *http.Client
}

func NewHTTPSender(store LocalStorage, baseURL string) *HTTPSender {
	return &HTTPSender{
		store:   store,
		baseURL: strings.TrimRight(baseURL, "/"),
		Client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *HTTPSender) sendMetric(ctx context.Context, mType, name, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path, err := url.JoinPath(s.baseURL, "update", mType, url.PathEscape(name), value)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, path)
	}

	return nil
}

func (s *HTTPSender) ReportOnce(ctx context.Context) error {
	gauges, counters := s.store.Snapshot()

	for k, v := range gauges {
		if err := s.sendMetric(ctx, model.Gauge, k, strconv.FormatFloat(v, 'f', -1, 64)); err != nil {
			return err
		}
	}

	for k, v := range counters {
		if err := s.sendMetric(ctx, model.Counter, k, strconv.FormatInt(v, 10)); err != nil {
			return err
		}
	}

	return nil
}
