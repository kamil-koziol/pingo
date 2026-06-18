package monitoring

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

type Monitor struct {
	URL            *url.URL
	Interval       time.Duration
	Name           string
	ExpectedStatus int
}

func NewCID() string {
	var b [16]byte

	_, err := rand.Read(b[:])
	if err != nil {
		// extremely rare; fallback to empty-ish safe value
		return "00000000000000000000000000000000"
	}

	return hex.EncodeToString(b[:])
}

func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.Interval)

	slog.InfoContext(ctx, "start monitoring",
		"name", m.Name,
		"url", m.URL.String(),
		"interval", m.Interval,
	)

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "monitor for finished", "url", m.URL.String())
			return
		case <-ticker.C:
			cid := NewCID()
			log := slog.Default()
			log = log.With("cid", cid)

			log.InfoContext(ctx, "checking", "url", m.URL.String())

			start := time.Now()
			resp, err := m.Call(ctx)
			if err != nil {
				log.ErrorContext(ctx, "there was an error during request", "err", err)
				continue
			}

			log.InfoContext(ctx, "got a response",
				"code", resp.StatusCode,
				"pass", resp.StatusCode == m.ExpectedStatus,
				"latency", time.Since(start),
			)
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
