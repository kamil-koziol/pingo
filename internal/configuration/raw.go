package configuration

import "gopkg.in/yaml.v3"

type rawCheck struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	Interval       string `yaml:"interval"`
	ExpectedStatus int    `yaml:"expected_status"`
}

type rawAlert struct {
	Type   string    `yaml:"type"`
	Name   string    `yaml:"name"`
	Config yaml.Node `yaml:"config"`
}

type rawConfig struct {
	Checks []rawCheck `yaml:"checks"`
	Alerts []rawAlert `yaml:"alerts"`
}
