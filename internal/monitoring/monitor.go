package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Monitor struct {
	URL      *url.URL
	Interval time.Duration
}

func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.Interval)

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "monitor for finished", "url", m.URL.String())
			return
		case <-ticker.C:
			slog.InfoContext(ctx, "checking", "url", m.URL.String())

			start := time.Now()
			resp, err := m.Call(ctx)
			if err != nil {
				slog.ErrorContext(ctx, "there was an error during request", "err", err)
				continue
			}

			slog.InfoContext(ctx, "got a response", "code", resp.StatusCode, "latency", time.Since(start))
		}
	}

}

func (m *Monitor) Call(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	return http.DefaultClient.Do(req)
}
