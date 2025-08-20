package main

import (
	"context"
	"strconv"

	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/linechart"
)

func RunTUI(ctx context.Context, metricCh <-chan map[string]float64, cfg *Config) error {
	lc, err := linechart.New(linechart.AxesCellOpts(cell.FgColor(cell.ColorWhite)))
	if err != nil {
		return err
	}

	t, err := termbox.New()
	if err != nil {
		return err
	}
	defer t.Close()

	cont, err := container.New(
		t,
		container.Border(linestyle.Light),
		container.BorderTitle("MongoDB serverStatus"),
		container.PlaceWidget(lc),
	)
	if err != nil {
		return err
	}

	go func() {
		series := make(map[string][]float64)
		for vals := range metricCh {
			for _, m := range cfg.Metrics {
				series[m.Name] = append(series[m.Name], vals[m.Name])
				if len(series[m.Name]) > 200 {
					series[m.Name] = series[m.Name][1:]
				}
				// metric color is stored as a string in the config; try to parse
				colorInt := int(cell.ColorWhite)
				if ci, err := strconv.Atoi(m.Color); err == nil {
					colorInt = ci
				}
				_ = lc.Series(m.Name, series[m.Name],
					linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(colorInt))),
				)
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
