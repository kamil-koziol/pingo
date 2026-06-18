package configuration

import (
	"fmt"
	"io"
	"net/url"
	"time"

	"gopkg.in/yaml.v3"
)

type Check struct {
	Name           string
	URL            *url.URL
	Interval       time.Duration
	ExpectedStatus int
}

type Config struct {
	Checks []Check
}

func Parse(r io.Reader) (*Config, error) {
	var raw rawConfig
	if err := yaml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}

	config := &Config{
		Checks: make([]Check, 0, len(raw.Checks)),
	}

	for _, c := range raw.Checks {
		u, err := url.Parse(c.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid url %q: %w", c.URL, err)
		}

		d, err := time.ParseDuration(c.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval %q: %w", c.Interval, err)
		}

		config.Checks = append(config.Checks, Check{
			Name:           c.Name,
			URL:            u,
			Interval:       d,
			ExpectedStatus: c.ExpectedStatus,
		})
	}

	return config, nil

}
