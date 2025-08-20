# mongospy

`mongospy` is a cross-platform terminal UI (TUI) tool for visualizing MongoDB [`serverStatus`](https://www.mongodb.com/docs/manual/reference/command/serverStatus/) metrics in real time.
It connects to a MongoDB instance, extracts configured fields, and charts them as time series graphs directly in your terminal.

## Features

- Live time-series charts in your terminal
- Connects directly via MongoDB Go driver (or via stdin for testing)
- Configurable metrics with dot-paths into `serverStatus`
- Counter → per-second rate computation (bytes/s, ops/s, etc.)
- Handles counter resets (e.g., server restarts)
- Works on Linux, macOS, and Windows
- Single static binary (no dependencies)
- One chart per metric: each row shows the metric name/value on the left and a small chart of recent changes on the right
- Values shown are deltas between samples (or rates when `derive: rate_per_sec` is configured)
- TUI title updates to include the `serverStatus.host` value for the connected server

## Installation

### Build from source

Requires Go 1.23+.

```bash
git clone https://github.com/alexbevi/mongospy.git
cd mongospy
go build -o mongospy
````

## Usage

Run with a config file:

```bash
./mongospy --config mongospy.yaml
```

Quick start with flags:

```bash
./mongospy \
  --uri mongodb://localhost:27017 \
  --interval 2s --window 5m
```

## Configuration

`mongospy` is configured using a YAML file. Example:

```yaml
uri: "mongodb://localhost:27017"
refreshInterval: "2s"
window: "10m"
metrics:
  - name: "bytesIn"
    path: "network.bytesIn"
    type: "counter"
    derive: "rate_per_sec"
    color: "2"
  - name: "bytesOut"
    path: "network.bytesOut"
    type: "counter"
    derive: "rate_per_sec"
    color: "3"
  - name: "snappyIn"
    path: "network.compression.snappy.bytesIn"
    type: "counter"
    derive: "rate_per_sec"
    color: "4"
  - name: "snappyOut"
    path: "network.compression.snappy.bytesOut"
    type: "counter"
    derive: "rate_per_sec"
    color: "5"
```

### Metric fields

* **`name`**: Label used in the legend
* **`path`**: Dot-path into `serverStatus` (e.g. `network.bytesIn`)
* **`type`**: `counter` (monotonically increasing) or `gauge` (instantaneous)
* **`derive`**: `none`, `rate_per_sec`, or `delta`
* **`color`**: Numeric or named color supported by the TUI

## Comparisons

You can compare two counters in the UI by adding a `compare` field to a metric
and referencing another metric by `name`. When present, the TUI will compute
running cumulative totals for both metrics and show a compact comparison in the
left-hand legend for the metric.

Example (compare `bytesIn` to `zlibIn`):

```yaml
- name: "bytesIn"
  path: "network.bytesIn"
  type: "counter"
  derive: "rate_per_sec"
  color: "2"
  compare: "zlibIn"

- name: "zlibIn"
  path: "network.compression.zlib.compressor.bytesIn"
  type: "counter"
  derive: "rate_per_sec"
  color: "4"
```

Display example in the legend:

```
bytesIn
1.02 MB
vs zlibIn: 1.02 MB / 21.86 MB (4.46%) since 1m2s
```

Notes:
- The percentage shown is this metric's share of the combined cumulative total
  (cumA / (cumA + cumB) * 100), which keeps the value bounded and easy to
  interpret.
- Cumulative totals are calculated in the UI from the sampled values; if a
  metric uses `derive: rate_per_sec` the sampled rate is converted back to a
  per-sample increment using the sampling interval before adding to the
  cumulative total.

## Keyboard Shortcuts

* `q` → Quit
* `p` → Pause sampling
* `e` → Export buffer to CSV/JSON (planned)
* `1..9` → Toggle metric visibility
* `TAB` → Cycle chart layouts

## License

MIT