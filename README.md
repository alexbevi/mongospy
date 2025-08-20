# mongospy

`mongospy` is a cross-platform terminal UI (TUI) tool for visualizing MongoDB [`serverStatus`](https://www.mongodb.com/docs/manual/reference/command/serverStatus/) metrics in real time.
It connects to a MongoDB instance, extracts configured fields, and charts them as time series graphs directly in your terminal.

---

## Features

- � Live time-series charts in your terminal
- � Connects directly via MongoDB Go driver (or via stdin for testing)
- ⚙️ Configurable metrics with dot-paths into `serverStatus`
- � Counter → per-second rate computation (bytes/s, ops/s, etc.)
- � Handles counter resets (e.g., server restarts)
- �️ Works on Linux, macOS, and Windows
- � Single static binary (no dependencies)

---

## Installation

### Build from source

Requires Go 1.23+.

```bash
git clone https://github.com/alexbevi/mongospy.git
cd mongospy
go build -o mongospy
````

---

## Usage

Run with a config file:

```bash
./mongospy --config mongospy.yaml
```

Quick start with flags:

```bash
./mongospy \
  --uri mongodb://localhost:27017 \
  --paths network.bytesIn,network.bytesOut \
  --interval 2s --window 5m
```

Testing via `mongosh` output:

```bash
mongosh "mongodb://localhost" --quiet \
  --eval "JSON.stringify(db.getSiblingDB('admin').serverStatus())" \
| ./mongospy --source stdin --paths network.bytesIn,network.bytesOut
```

---

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

---

## Keyboard Shortcuts

* `q` → Quit
* `p` → Pause sampling
* `e` → Export buffer to CSV/JSON (planned)
* `1..9` → Toggle metric visibility
* `TAB` → Cycle chart layouts

---

## Example

Tracking network traffic:

```
bytesIn    ──────────╮
bytesOut   ╮─────────╯
snappyIn       ╭─────
snappyOut      ╯
```

Legend shows the latest values (e.g. `bytesIn: 12.3 MB/s`).

---

## Roadmap

* [ ] Export collected data as CSV/JSON
* [ ] Multiple chart panels (per subsystem)
* [ ] Derived metrics (e.g., compression savings)
* [ ] Threshold alerts with color flips
* [ ] Replay mode from saved JSONL

---

## Requirements & Permissions

* Needs a user with `serverStatus` privilege on the `admin` database
* Works against standalone, replica set, and sharded cluster nodes
* For cluster-wide stats, run separately against each node

---

## License

MIT