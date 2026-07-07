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
	"net/http"
	"time"

	"github.com/kamil-koziol/pingo/internal/alerting"
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
	log := slog.Default()
	log = log.With("cid", cid)

	log.InfoContext(ctx, "checking", "url", m.service.Url, "service", m.service.Name)

	start := time.Now()
	resp, err := m.Call(ctx)
	if err != nil {
		log.ErrorContext(ctx, "there was an error during request", "err", err)
		return fmt.Errorf("unable to call: %w", err)
	}

	latency := time.Since(start)
	isUp := int64(resp.StatusCode) == m.service.ExpectedStatus

	log.InfoContext(ctx, "got a response",
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
		if isUp {
			m.alerter.Publish(ctx, &alerting.ServiceRecoveredEvent{
				ServiceId:   m.service.ID,
				ServiceName: m.service.Name,
				StatusCode:  int32(resp.StatusCode),
				LatencyMs:   latency.Milliseconds(),
			})
		} else {
			m.alerter.Publish(ctx, &alerting.ServiceDownEvent{
				ServiceId:      m.service.ID,
				ServiceName:    m.service.Name,
				StatusCode:     int32(resp.StatusCode),
				ExpectedStatus: int32(m.service.ExpectedStatus),
				LatencyMs:      latency.Milliseconds(),
				Error:          string(b),
			})
		}
	}

	return nil
}
