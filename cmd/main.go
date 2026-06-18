package main

import (
	"context"
	"log"
	"os"

	"github.com/kamil-koziol/pingo/internal/configuration"
	"github.com/kamil-koziol/pingo/internal/monitoring"
)

func main() {
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalf("unable to read config: %v", err)
	}

	config, err := configuration.Parse(f)
	if err != nil {
		log.Fatalf("unable to parse config: %v", err)
	}

	ctx := context.Background()
	for _, check := range config.Checks {
		m := monitoring.Monitor{
			URL:            check.URL,
			Interval:       check.Interval,
			Name:           check.Name,
			ExpectedStatus: check.ExpectedStatus,
		}

		go m.Run(ctx)
	}

	<-ctx.Done()
}
