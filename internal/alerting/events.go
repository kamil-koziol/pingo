package alerting

type EventType string

const (
	EventTypeServiceDown      EventType = "service_down"
	EventTypeServiceRecovered EventType = "service_recovered"
)

type ServiceRecoveredEvent struct {
	ServiceId   int64
	ServiceName string
	StatusCode  int32
	LatencyMs   int64
}

func (e *ServiceRecoveredEvent) Type() EventType {
	return EventTypeServiceRecovered
}

type ServiceDownEvent struct {
	ServiceId      int64
	ServiceName    string
	StatusCode     int32
	ExpectedStatus int32
	LatencyMs      int64
	Error          string
}

func (e *ServiceDownEvent) Type() EventType {
	return EventTypeServiceDown
}
