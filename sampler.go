package main

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RunSampler(cfg *Config, out chan<- map[string]float64) error {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return err
	}

	db := client.Database("admin")

	tick, _ := time.ParseDuration(cfg.RefreshInterval)

	go func() {
		for {
			select {
			case <-time.After(tick):
				var result bson.M
				if err := db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result); err != nil {
					continue
				}

				values := map[string]float64{}
				for _, m := range cfg.Metrics {
					if v, ok := resolvePath(result, m.Path); ok {
						values[m.Name] = v
					}
				}
				out <- values
			}
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
