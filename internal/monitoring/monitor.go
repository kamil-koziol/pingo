package monitoring

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	randv2 "math/rand/v2"
	"net/http"
	"time"

	"github.com/kamil-koziol/pingo/internal/alerting"
	"github.com/kamil-koziol/pingo/internal/contextx"
	"github.com/kamil-koziol/pingo/internal/db"
)

type Monitor struct {
	service *db.Service
	q       *db.Queries
	alerter alerting.Alerter
}

func NewMonitor(service *db.Service, queries *db.Queries, alerter alerting.Alerter) *Monitor {
	return &Monitor{
		service: service,
		q:       queries,
		alerter: alerter,
	}
}

func newCID() string {
	var b [16]byte

	_, err := rand.Read(b[:])
	if err != nil {
		// extremely rare; fallback to empty-ish safe value
		return "00000000000000000000000000000000"
	}

	return hex.EncodeToString(b[:])
}

func applyJitter(d time.Duration) time.Duration {
	multiplier := 0.9 + randv2.Float64()*0.2 // 0.9 <= x < 1.1
	return time.Duration(float64(d) * multiplier)
}

func (m *Monitor) Run(ctx context.Context) {
	interval := time.Duration(m.service.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)

	slog.InfoContext(ctx, "start monitoring",
		"name", m.service.Name,
		"url", m.service.Url,
		"interval", interval,
	)

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "monitor for finished", "url", m.service.Url)
			return
		case <-ticker.C:
			if err := m.Ping(ctx); err != nil {
				slog.ErrorContext(ctx, "failure during ping", "err", err)
			}
			ticker.Reset(applyJitter(interval))
		}
	}
}

func (m *Monitor) Call(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.service.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	return http.DefaultClient.Do(req)
}

func (m *Monitor) Ping(ctx context.Context) error {
	cid := newCID()

	logger := contextx.Logger(ctx)
	logger = logger.With("cid", cid)

	logger.InfoContext(ctx, "checking", "url", m.service.Url, "service", m.service.Name)

	start := time.Now()
	resp, err := m.Call(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "there was an error during request", "err", err)
		return fmt.Errorf("unable to call: %w", err)
	}

	latency := time.Since(start)
	isUp := int64(resp.StatusCode) == m.service.ExpectedStatus

	logger.InfoContext(ctx, "got a response",
		"code", resp.StatusCode,
		"is_up", isUp,
		"latency", latency,
	)

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read the body: %w", err)
	}

	if err = m.q.CreatePing(ctx, db.CreatePingParams{
		ServiceID:          m.service.ID,
		StatusCode:         int64(resp.StatusCode),
		ExpectedStatusCode: int64(m.service.ExpectedStatus),
		LatencyMs:          latency.Milliseconds(),
		IsUp:               isUp,
		ErrorMessage:       sql.NullString{String: string(b), Valid: !isUp},
		Timestamp:          time.Now(),
	}); err != nil {
		return fmt.Errorf("unable to create ping: %w", err)
	}

	latestPing, err := m.q.GetLatestServicePing(ctx, m.service.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			latestPing = nil
		} else {
			return fmt.Errorf("unable to get latest ping: %w", err)
		}
	}

	if latestPing != nil && latestPing.IsUp != isUp {
		var event alerting.Event

		if isUp {
			event = &alerting.ServiceRecoveredEvent{
				ServiceId:   m.service.ID,
				ServiceName: m.service.Name,
				StatusCode:  int32(resp.StatusCode),
				LatencyMs:   latency.Milliseconds(),
			}
		} else {
			event = &alerting.ServiceDownEvent{
				ServiceId:      m.service.ID,
				ServiceName:    m.service.Name,
				StatusCode:     int32(resp.StatusCode),
				ExpectedStatus: int32(m.service.ExpectedStatus),
				LatencyMs:      latency.Milliseconds(),
				Error:          string(b),
			}
		}

		if err := m.alerter.Publish(ctx, event); err != nil {
			logger.ErrorContext(ctx, "publishing events failed", "err", err)
		}
	}

	return nil
}
