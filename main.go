package main

import (
	"context"
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "mongospy.yaml", "path to config")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	metricCh := make(chan map[string]float64, 1)
	hostCh := make(chan string, 1)
	if err := RunSampler(cfg, metricCh, hostCh); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := RunTUI(ctx, metricCh, cfg, hostCh); err != nil {
		log.Fatal(err)
	}
}
