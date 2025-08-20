package main

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RunSampler(cfg *Config, out chan<- map[string]float64, hostCh chan<- string) error {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return err
	}

	db := client.Database("admin")

	tick, _ := time.ParseDuration(cfg.RefreshInterval)

	go func() {
		prev := make(map[string]float64)
		tickSeconds := tick.Seconds()
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		prev = make(map[string]float64)
		tickSeconds = tick.Seconds()
		for range ticker.C {
				var result bson.M
				if err := db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result); err != nil {
					continue
				}

				// extract host field if present
				if host, ok := result["host"].(string); ok {
					select {
					case hostCh <- host:
					default:
					}
				}

				values := map[string]float64{}
				for _, m := range cfg.Metrics {
					if cur, ok := resolvePath(result, m.Path); ok {
						// if metric is a counter (or derive requests deltas), compute delta
						if m.Type == "counter" || m.Derive == "delta" || m.Derive == "rate_per_sec" {
							if prevV, ok := prev[m.Name]; ok {
								delta := cur - prevV
								if delta < 0 {
									// counter reset detected; treat as cur value
									delta = cur
								}
								if m.Derive == "rate_per_sec" {
									values[m.Name] = delta / tickSeconds
								} else {
									values[m.Name] = delta
								}
							} else {
								// no previous sample; emit 0
								values[m.Name] = 0
							}
						} else {
							// gauges and others: pass current value
							values[m.Name] = cur
						}
						prev[m.Name] = cur
					}
				}
				out <- values
		}
	}()

	return nil
}

// resolvePath walks a dot.path into a bson.M
func resolvePath(doc any, path string) (float64, bool) {
	parts := strings.Split(path, ".")
	cur := doc
	for _, p := range parts {
		if m, ok := cur.(bson.M); ok {
			cur = m[p]
		} else {
			return 0, false
		}
	}
	switch v := cur.(type) {
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
