package configuration

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/kamil-koziol/pingo/internal/alerting"
	"gopkg.in/yaml.v3"
)

type Check struct {
	Name           string
	URL            *url.URL
	Interval       time.Duration
	ExpectedStatus int
}

type AlerterConfig interface {
	Build() (alerting.Alerter, error)
}

type Config struct {
	API    *rawAPI
	Checks []Check
	Alerts []AlerterConfig
}

type TelegramConfig struct {
	Name     string
	BotToken string
	ChatID   string
}

func (t *TelegramConfig) Build() (alerting.Alerter, error) {
	return &alerting.TelegramAlerter{
		BotToken: t.BotToken,
		ChatID:   t.ChatID,
	}, nil
}

func Parse(r io.Reader) (*Config, error) {
	var raw rawConfig
	if err := yaml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	raw.Default()

	if err := raw.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	config := &Config{
		Checks: make([]Check, 0, len(raw.Checks)),
		Alerts: make([]AlerterConfig, 0, len(raw.Alerts)),
		API:    raw.API,
	}

	for _, c := range raw.Checks {
		u, err := url.Parse(c.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid url %q: %w", c.URL, err)
		}

		config.Checks = append(config.Checks, Check{
			Name:           c.Name,
			URL:            u,
			Interval:       c.Interval,
			ExpectedStatus: c.ExpectedStatus,
		})
	}

	for _, a := range raw.Alerts {
		switch a.Type {
		case "telegram":
			var cfg struct {
				BotToken string `yaml:"bot_token"`
				ChatID   string `yaml:"chat_id"`
			}

			if err := a.Config.Decode(&cfg); err != nil {
				return nil, err
			}

			config.Alerts = append(config.Alerts, &TelegramConfig{
				Name:     a.Name,
				BotToken: cfg.BotToken,
				ChatID:   cfg.ChatID,
			})
		default:
			return nil, fmt.Errorf("unsupported type: %s", a.Type)
		}
	}

	return config, nil
}
