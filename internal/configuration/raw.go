package configuration

import (
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"gopkg.in/yaml.v3"
)

type rawHTTP struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func (h *rawHTTP) Default() {
	if h.Enabled && h.Port == 0 {
		h.Port = 8080
	}
}

func (h *rawHTTP) Validate() error {
	return validation.ValidateStruct(h,
		validation.Field(&h.Port,
			validation.When(h.Enabled, validation.Min(1), validation.Max(65535)),
		),
	)
}

type rawGRPC struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func (g *rawGRPC) Default() {
	if g.Enabled && g.Port == 0 {
		g.Port = 50051
	}
}

func (g *rawGRPC) Validate() error {
	return validation.ValidateStruct(g,
		validation.Field(&g.Port,
			validation.When(g.Enabled, validation.Min(1), validation.Max(65535)),
		),
	)
}

type rawAPI struct {
	HTTP *rawHTTP `yaml:"http"`
	GRPC *rawGRPC `yaml:"grpc"`
}

func (c *rawAPI) Default() {
	if c.HTTP == nil {
		c.HTTP = &rawHTTP{}
	}
	if c.GRPC == nil {
		c.GRPC = &rawGRPC{}
	}

	c.HTTP.Default()
	c.GRPC.Default()
}

func (c *rawAPI) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.HTTP),
		validation.Field(&c.GRPC),

		// HTTP cannot be enabled if gRPC is disabled
		validation.Field(&c.HTTP,
			validation.By(func(value interface{}) error {
				httpEnabled := c.HTTP != nil && c.HTTP.Enabled
				grpcEnabled := c.GRPC != nil && c.GRPC.Enabled

				if httpEnabled && !grpcEnabled {
					return fmt.Errorf("cannot be enabled when gRPC is disabled")
				}
				return nil
			}),
		),
	)
}

type rawCheck struct {
	Name           string        `yaml:"name"`
	URL            string        `yaml:"url"`
	Interval       time.Duration `yaml:"interval"`
	ExpectedStatus int           `yaml:"expected_status"`
}

func (c *rawCheck) Default() {
	if c.Interval == 0 {
		c.Interval = 10 * time.Second
	}
	if c.ExpectedStatus == 0 {
		c.ExpectedStatus = 200
	}
}

func (c *rawCheck) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.URL, validation.Required, is.URL),
		validation.Field(&c.Interval, validation.Min(time.Second)),
		validation.Field(&c.ExpectedStatus, validation.Min(100), validation.Max(599)),
	)
}

type rawAlert struct {
	Type   string    `yaml:"type"`
	Name   string    `yaml:"name"`
	Config yaml.Node `yaml:"config"`
}

func (a *rawAlert) Default() {
	// No defaults needed for rawAlert currently
}

func (a *rawAlert) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Type, validation.Required),
		validation.Field(&a.Name, validation.Required),
		validation.Field(&a.Config, validation.By(validateYAMLNodeNotEmpty)),
	)
}

type rawConfig struct {
	API    *rawAPI     `yaml:"api"`
	Checks []*rawCheck `yaml:"checks"`
	Alerts []*rawAlert `yaml:"alerts"`
}

func (c *rawConfig) Default() {
	if c.API == nil {
		c.API = &rawAPI{}
	}
	c.API.Default()

	for _, check := range c.Checks {
		if check != nil {
			check.Default()
		}
	}
	for _, alert := range c.Alerts {
		if alert != nil {
			alert.Default()
		}
	}
}

func (c *rawConfig) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.API),
		validation.Field(&c.Checks, validation.Each()),
		validation.Field(&c.Alerts, validation.Each()),
	)
}

// Custom validator to check if yaml.Node is not zero/empty
func validateYAMLNodeNotEmpty(value interface{}) error {
	node, ok := value.(yaml.Node)
	if !ok || node.IsZero() {
		return fmt.Errorf("cannot be empty")
	}
	return nil
}
