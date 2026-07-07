package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type TelegramAlerter struct {
	BotToken string
	ChatID   string
}

func (t *TelegramAlerter) getTextForEvent(event Event) (string, error) {
	switch e := event.(type) {
	case *ServiceDownEvent:
		return fmt.Sprintf("%s: is down! received %d, expected %d", e.ServiceName, e.StatusCode, e.ExpectedStatus), nil
	case *ServiceRecoveredEvent:
		return fmt.Sprintf("%s: recovered", e.ServiceName), nil
	default:
		return "", ErrUnsupported
	}
}

func (t *TelegramAlerter) Publish(ctx context.Context, event Event) error {
	text, err := t.getTextForEvent(event)
	if err != nil {
		return fmt.Errorf("unable to get message for event: %w", err)
	}

	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage",
		t.BotToken,
	)

	payload := map[string]string{
		"chat_id": t.ChatID,
		"text":    text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram error: %s", resp.Status)
	}

	return nil
}
