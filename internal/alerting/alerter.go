package alerting

import (
	"context"
	"errors"
)

type Event interface {
	Type() EventType
}

var ErrUnsupported = errors.New("alert type not supported by this backend")

type Alerter interface {
	Publish(context.Context, Event) error
}
