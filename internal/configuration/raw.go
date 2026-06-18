package configuration

type rawCheck struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	Interval       string `yaml:"interval"`
	ExpectedStatus int    `yaml:"expected_status"`
}

type rawConfig struct {
	Checks []rawCheck `yaml:"checks"`
}
