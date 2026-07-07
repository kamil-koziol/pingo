package alerting

import (
	"context"
	"errors"
)

type MultiAlerter struct {
	Alerters []Alerter
}

func NewMultiAlerter(alerters ...Alerter) *MultiAlerter {
	return &MultiAlerter{
		Alerters: alerters,
	}
}

func (m *MultiAlerter) Publish(ctx context.Context, event Event) error {
	for _, alerter := range m.Alerters {
		err := alerter.Publish(ctx, event)
		if err != nil {
			if errors.Is(err, ErrUnsupported) {
				continue
			}
			return err
		}
	}
	return nil
}
