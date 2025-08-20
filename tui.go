package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/text"
)

func RunTUI(ctx context.Context, metricCh <-chan map[string]float64, cfg *Config, hostCh <-chan string) error {
	// per-metric charts created below

	t, err := termbox.New()
	if err != nil {
		return err
	}
	defer t.Close()

	// create per-metric widgets
	n := len(cfg.Metrics)
	if n == 0 {
		return fmt.Errorf("no metrics configured")
	}

	texts := make([]*text.Text, n)
	charts := make([]*linechart.LineChart, n)
	for i := range cfg.Metrics {
		txtw, err := text.New()
		if err != nil {
			return err
		}
		texts[i] = txtw

		lcw, err := linechart.New(
			linechart.AxesCellOpts(cell.FgColor(cell.ColorWhite)),
			// Hide Y axis labels so all charts reserve the same minimal left space
			linechart.YAxisFormattedValues(func(_ float64) string { return "" }),
		)
		if err != nil {
			return err
		}
		charts[i] = lcw
	}

	// helper to create a SplitVertical option for a single metric row
	makeRow := func(i int) container.Option {
		return container.SplitVertical(
			container.Left(container.PlaceWidget(texts[i])),
			container.Right(container.PlaceWidget(charts[i])),
			container.SplitPercent(25),
		)
	}

	// build nested SplitHorizontal tree to stack rows vertically
	var rootSplit container.Option
	// Start from the last metric and nest upwards so the top is metric 0
	for i := n - 1; i >= 0; i-- {
		if i == n-1 {
			// last row: top = row(i), bottom = nothing -> just use Top(row)
			rootSplit = container.SplitHorizontal(
				container.Top(makeRow(i)),
				container.Bottom(),
				container.SplitPercent(100/(n)),
			)
		} else {
			// nest previous rootSplit into bottom
			rootSplit = container.SplitHorizontal(
				container.Top(makeRow(i)),
				container.Bottom(rootSplit),
				container.SplitPercent(100/(n)),
			)
		}
	}

	cont, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.ID("root"),
		container.BorderTitle("MongoDB serverStatus"),
		rootSplit,
	)
	if err != nil {
		return err
	}

	// listen for host value and update title
	go func() {
		for h := range hostCh {
			_ = cont.Update("root", container.BorderTitle("MongoDB serverStatus for "+h))
		}
	}()

	go func() {
		series := make(map[string][]float64)
		// timestamps aligned with samples
		timestamps := make([]time.Time, 0, 200)
		var startTime time.Time
		// cumulative totals per metric (running total since UI started)
		cumValues := make(map[string]float64)
		for vals := range metricCh {
			// append timestamp
			timestamps = append(timestamps, time.Now())
			if startTime.IsZero() {
				startTime = time.Now()
			}
			if len(timestamps) > 200 {
				timestamps = timestamps[1:]
			}

			// compute intervalSeconds based on last two timestamps when possible
			intervalSeconds := 0.0
			if len(timestamps) >= 2 {
				intervalSeconds = timestamps[len(timestamps)-1].Sub(timestamps[len(timestamps)-2]).Seconds()
			}

			for i, m := range cfg.Metrics {
				series[m.Name] = append(series[m.Name], vals[m.Name])
				if len(series[m.Name]) > 200 {
					series[m.Name] = series[m.Name][1:]
				}

				// update cumulative value for this metric using per-sample value
				// If metric is a per-second rate, convert back to per-sample by multiplying by intervalSeconds.
				perSample := vals[m.Name]
				if m.Derive == "rate_per_sec" && intervalSeconds > 0 {
					perSample = vals[m.Name] * intervalSeconds
				}
				// For gauges or when interval is unknown, treat perSample as the emitted value.
				cumValues[m.Name] += perSample

				// metric color is stored as a string in the config; try to parse
				colorInt := int(cell.ColorWhite)
				if ci, err := strconv.Atoi(m.Color); err == nil {
					colorInt = ci
				}

				// build x labels (first, middle, last) using timestamps
				labels := make(map[int]string)
				ln := len(series[m.Name])
				if ln > 0 {
					// map positions relative to current series length
					// assume timestamps length == series length
					if len(timestamps) >= ln {
						if ln >= 1 {
							labels[0] = timestamps[len(timestamps)-ln].Format("15:04:05")
						}
						if ln >= 2 {
							mid := ln / 2
							labels[mid] = timestamps[len(timestamps)-ln+mid].Format("15:04:05")
						}
						if ln >= 3 {
							labels[ln-1] = timestamps[len(timestamps)-1].Format("15:04:05")
						}
					}
				}

				// update chart for this metric
				_ = charts[i].Series(m.Name, series[m.Name],
					linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(colorInt))),
					linechart.SeriesXLabels(labels),
				)

				// update left text widget for this metric (name + formatted latest)
				latest := series[m.Name][len(series[m.Name])-1]
				legend := m.Name + "\n" + formatValue(m, latest) + "\n"
				// if this metric compares to another, display the running total delta and percentage
				if m.Compare != "" {
					if _, ok := series[m.Compare]; ok {
						// cumulative values for this metric and the compared metric
						cumA := cumValues[m.Name]
						cumB := cumValues[m.Compare]
						pctStr := "N/A"
						total := cumA + cumB
						if total != 0 {
							// show this metric's share of the combined total (bounded 0-100%)
							pct := (cumA / total) * 100.0
							pctStr = fmt.Sprintf("%.2f%%", pct)
						}
						// show cumulative totals for both metrics (formatted consistently)
						left := formatCumulative(m, cumA)
						right := formatCumulative(m, cumB)
						elapsed := time.Since(startTime)
						legend += fmt.Sprintf("vs %s: %s / %s (%s) since %s\n", m.Compare, left, right, pctStr, formatDuration(elapsed))
					}
				}

				_ = texts[i].Write(legend, text.WriteReplace())
			}
		}
	}()

	// create a cancellable context so we can stop termdash when user presses 'q'
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	kbSub := func(k *terminalapi.Keyboard) {
		// printable characters are their rune value
		switch k.Key {
		case 'q', 'Q', '\r':
			cancel()
		case keyboard.KeyCtrlC, keyboard.KeyEsc:
			cancel()
		}
	}

	return termdash.Run(runCtx, t, cont, termdash.KeyboardSubscriber(kbSub))
}

// formatValue returns a human-friendly string for the metric value.
// - If the metric path or name contains "byte" or "bytes" we render in human bytes.
// - If metric Derive is "rate_per_sec" we append "/s".
// - Otherwise we format with two decimals.
func formatValue(m MetricConfig, v float64) string {
	isBytes := false
	if m.Path != "" {
		p := strings.ToLower(m.Path)
		if strings.Contains(p, "byte") || strings.Contains(p, "bytes") {
			isBytes = true
		}
	}
	if !isBytes {
		n := strings.ToLower(m.Name)
		if strings.Contains(n, "byte") || strings.Contains(n, "bytes") {
			isBytes = true
		}
	}

	var s string
	if isBytes {
		s = humanBytes(v)
	} else {
		s = fmt.Sprintf("%.2f", v)
	}
	if m.Derive == "rate_per_sec" {
		s += "/s"
	}
	return s
}

// formatCumulative formats a cumulative total for display in comparisons.
// It uses the same byte detection heuristics as formatValue but does not
// append "/s" even if the metric has Derive == "rate_per_sec".
func formatCumulative(m MetricConfig, v float64) string {
	isBytes := false
	if m.Path != "" {
		p := strings.ToLower(m.Path)
		if strings.Contains(p, "byte") || strings.Contains(p, "bytes") {
			isBytes = true
		}
	}
	if !isBytes {
		n := strings.ToLower(m.Name)
		if strings.Contains(n, "byte") || strings.Contains(n, "bytes") {
			isBytes = true
		}
	}

	if isBytes {
		return humanBytes(v)
	}
	return fmt.Sprintf("%.2f", v)
}

// humanBytes formats a float64 number of bytes into a human readable string.
func humanBytes(b float64) string {
	if math.IsNaN(b) || math.IsInf(b, 0) {
		return "NaN"
	}
	if b < 1024 {
		return fmt.Sprintf("%.0f B", b)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	val := b / 1024.0
	i := 0
	for ; i < len(units)-1 && val >= 1024.0; i++ {
		val /= 1024.0
	}
	return fmt.Sprintf("%.2f %s", val, units[i])
}

// formatDuration returns a concise human readable duration like "1h2m3s".
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}
	// drop fractional seconds
	secs := int64(d.Seconds())
	h := secs / 3600
	secs %= 3600
	m := secs / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
