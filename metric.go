package main

import "time"

type Sample struct {
	Time  time.Time
	Value float64
}

type Metric struct {
	Config  MetricConfig
	Samples []Sample
	Prev    float64
}

func (m *Metric) AddSample(val float64) {
	now := time.Now()
	if len(m.Samples) > 200 {
		m.Samples = m.Samples[1:]
	}
	m.Samples = append(m.Samples, Sample{Time: now, Value: val})
	m.Prev = val
}
