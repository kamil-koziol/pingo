package main

import (
	"context"
	"net/url"
	"time"

	"github.com/kamil-koziol/pingo/internal/monitoring"
)

func main() {
	u, _ := url.Parse("https://kamilkoziol.com")
	m := monitoring.Monitor{
		URL:      u,
		Interval: time.Second,
	}

	ctx := context.Background()
	m.Run(ctx)
}
